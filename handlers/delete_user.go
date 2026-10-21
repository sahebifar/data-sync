package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"data-sync/database"
)

func DeleteUserHandler(postgresDB, yugabyteDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"success": false,
				"message": "Method not allowed; use DELETE",
			})
			return
		}

		id, err := parsePositiveID(r.URL.Query().Get("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "A valid user id is required",
			})
			return
		}

		tx, err := postgresDB.BeginTx(r.Context(), nil)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to start PostgreSQL transaction",
			})
			return
		}
		defer tx.Rollback()

		var email string
		err = tx.QueryRowContext(r.Context(), `
			SELECT email
			FROM users
			WHERE id = $1
			FOR UPDATE
		`, id).Scan(&email)

		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "User not found in PostgreSQL",
			})
			return
		}

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to find user in PostgreSQL",
			})
			return
		}

		result, err := tx.ExecContext(r.Context(), `
			DELETE FROM users
			WHERE id = $1
		`, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to delete user from PostgreSQL",
			})
			return
		}

		affected, err := result.RowsAffected()
		if err != nil || affected != 1 {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Could not confirm user deletion in PostgreSQL",
			})
			return
		}

		// Store the delete operation in the same transaction.
		if err := database.EnqueueSync(
			tx,
			email,
			"delete",
			map[string]interface{}{
				"email": email,
			},
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to queue user deletion",
			})
			return
		}

		if err := tx.Commit(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to commit user deletion and sync request",
			})
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"success": true,
			"message": "User deleted from PostgreSQL and queued for synchronization",
			"data": map[string]interface{}{
				"email":       email,
				"sync_status": "pending",
			},
		})
	}
}
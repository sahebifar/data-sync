package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"data-sync/database"

	"github.com/lib/pq"
)

type updateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Age       int    `json:"age"`
}

func UpdateUserHandler(postgresDB, yugabyteDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"success": false,
				"message": "Method not allowed; use PUT",
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

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()

		var input updateUserRequest
		decoder := json.NewDecoder(r.Body)

		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Invalid JSON request body",
			})
			return
		}

		var extra interface{}
		if err := decoder.Decode(&extra); err != io.EOF {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Request must contain exactly one JSON object",
			})
			return
		}

		input.FirstName = strings.TrimSpace(input.FirstName)
		input.LastName = strings.TrimSpace(input.LastName)

		if input.FirstName == "" ||
			input.LastName == "" ||
			len(input.FirstName) > 100 ||
			len(input.LastName) > 100 ||
			input.Age < 1 ||
			input.Age > 120 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Valid first name, last name, and age between 1 and 120 are required",
			})
			return
		}

		if err := postgresDB.PingContext(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				"success": false,
				"message": "PostgreSQL is unavailable",
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

		var updated struct {
			ID        int64     `json:"id"`
			FirstName string    `json:"first_name"`
			LastName  string    `json:"last_name"`
			Email     string    `json:"email"`
			Age       int       `json:"age"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		}

		err = tx.QueryRowContext(r.Context(), `
			UPDATE users
			SET first_name = $1,
			    last_name = $2,
			    age = $3,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $4
			RETURNING id, first_name, last_name, email, age,
			          created_at, updated_at
		`,
			input.FirstName,
			input.LastName,
			input.Age,
			id,
		).Scan(
			&updated.ID,
			&updated.FirstName,
			&updated.LastName,
			&updated.Email,
			&updated.Age,
			&updated.CreatedAt,
			&updated.UpdatedAt,
		)

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
				"message": "Failed to update PostgreSQL",
			})
			return
		}

		payload := map[string]interface{}{
			"first_name": updated.FirstName,
			"last_name":  updated.LastName,
			"email":      updated.Email,
			"age":        updated.Age,
		}

		if err := database.EnqueueSync(
			tx,
			updated.Email,
			"update",
			payload,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to queue user synchronization",
			})
			return
		}

		if err := tx.Commit(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to commit user update and sync request",
			})
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]interface{}{
			"success": true,
			"message": "User updated in PostgreSQL and queued for synchronization",
			"data": map[string]interface{}{
				"user":        updated,
				"sync_status": "pending",
			},
		})
	}
}

// Keep the pq import available if future PostgreSQL-specific
// error handling is added here.
var _ *pq.Error

package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"data-sync/database"

	"github.com/lib/pq"
)

type createUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Age       *int    `json:"age"`
}

type createdUser struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func CreateUserHandler(
	postgresDB, yugabyteDB *sql.DB,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"success": false,
				"message": "Method not allowed",
			})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()

		var input createUserRequest
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

		if input.FirstName == nil ||
			input.LastName == nil ||
			input.Email == nil ||
			input.Age == nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "First name, last name, email and age are required",
			})
			return
		}

		firstName := strings.TrimSpace(*input.FirstName)
		lastName := strings.TrimSpace(*input.LastName)
		email := strings.ToLower(strings.TrimSpace(*input.Email))
		age := *input.Age

		if firstName == "" || lastName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "First name and last name are required",
			})
			return
		}

		if len(firstName) > 100 || len(lastName) > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Names must not exceed 100 characters",
			})
			return
		}

		if len(email) > 255 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Email must not exceed 255 characters",
			})
			return
		}

		parsedEmail, err := mail.ParseAddress(email)
		if err != nil || parsedEmail.Address != email {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "A valid email address is required",
			})
			return
		}

		if age < 1 || age > 120 {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Age must be between 1 and 120",
			})
			return
		}

		// PostgreSQL is the source of truth.
		// YugabyteDB is allowed to be temporarily unavailable.
		var ybAvailable bool
		if err := yugabyteDB.Ping(); err == nil {
			ybAvailable = true
		}

		var exists bool
		err = postgresDB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM users WHERE LOWER(email) = $1
			)
		`, email).Scan(&exists)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to check existing users in PostgreSQL",
			})
			return
		}

		if exists {
			writeJSON(w, http.StatusConflict, map[string]interface{}{
				"success": false,
				"message": "Email already exists in PostgreSQL",
			})
			return
		}

		// If YugabyteDB is reachable, prevent creating a duplicate there.
		if ybAvailable {
			err = yugabyteDB.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM users WHERE LOWER(email) = $1
				)
			`, email).Scan(&exists)

			if err != nil {
				// Don't write to PostgreSQL if the target's state
				// cannot be checked reliably.
				writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
					"success": false,
					"message": "Could not check existing users in YugabyteDB",
				})
				return
			}

			if exists {
				writeJSON(w, http.StatusConflict, map[string]interface{}{
					"success": false,
					"message": "Email already exists in YugabyteDB",
				})
				return
			}
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

		var pgUser createdUser
		err = tx.QueryRowContext(r.Context(), `
			INSERT INTO users (first_name, last_name, email, age)
			VALUES ($1, $2, $3, $4)
			RETURNING id, first_name, last_name, email, age,
			          created_at, updated_at
		`, firstName, lastName, email, age).Scan(
			&pgUser.ID,
			&pgUser.FirstName,
			&pgUser.LastName,
			&pgUser.Email,
			&pgUser.Age,
			&pgUser.CreatedAt,
			&pgUser.UpdatedAt,
		)

		if err != nil {
			var pgErr *pq.Error
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				writeJSON(w, http.StatusConflict, map[string]interface{}{
					"success": false,
					"message": "Email already exists",
				})
				return
			}

			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to create user in PostgreSQL",
			})
			return
		}

		// Save a durable sync request in the SAME transaction.
		payload := map[string]interface{}{
			"first_name": firstName,
			"last_name":  lastName,
			"email":      email,
			"age":        age,
		}

		if err := database.EnqueueSync(tx, email, "create", payload); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to queue user synchronization",
			})
			return
		}

		if err := tx.Commit(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to commit user and sync request",
			})
			return
		}

		// The user and the sync request are safely stored.
		// A worker will deliver the request to YugabyteDB later.
		status := http.StatusAccepted
		message := "User saved in PostgreSQL and queued for synchronization"

		if ybAvailable {
			message = "User saved in PostgreSQL and queued for synchronization"
		}

		writeJSON(w, status, map[string]interface{}{
			"success": true,
			"message": message,
			"data": map[string]interface{}{
				"postgres_user": pgUser,
				"sync_status":   "pending",
			},
		})
	}
}

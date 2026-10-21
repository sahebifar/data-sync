package handlers

import (
	"database/sql"
	"net/http"
	"strings"
)

func SyncMissingUsersHandler(postgresDB, yugabyteDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"success": false,
				"message": "Method not allowed",
			})
			return
		}

		sourceUsers, err := loadUsers(postgresDB)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to read PostgreSQL users",
			})
			return
		}

		targetUsers, err := loadUsers(yugabyteDB)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to read YugabyteDB users",
			})
			return
		}

		inserted := 0
		skipped := 0
		failures := make([]map[string]string, 0)

		for email, user := range sourceUsers {
			if _, exists := targetUsers[email]; exists {
				skipped++
				continue
			}

			if strings.TrimSpace(user.FirstName) == "" ||
				strings.TrimSpace(user.LastName) == "" ||
				strings.TrimSpace(user.Email) == "" ||
				user.Age == nil ||
				*user.Age <= 0 {

				failures = append(failures, map[string]string{
					"email":  email,
					"reason": "Source user has incomplete or invalid data",
				})
				continue
			}

			result, err := yugabyteDB.Exec(`
				INSERT INTO users (first_name, last_name, email, age)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (email) DO NOTHING
			`,
				user.FirstName,
				user.LastName,
				user.Email,
				*user.Age,
			)

			if err != nil {
				failures = append(failures, map[string]string{
					"email":  email,
					"reason": "Insert into YugabyteDB failed",
				})
				continue
			}

			affected, err := result.RowsAffected()
			if err != nil {
				failures = append(failures, map[string]string{
					"email":  email,
					"reason": "Could not verify insert result",
				})
				continue
			}

			if affected == 1 {
				inserted++
			} else {
				skipped++
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":  len(failures) == 0,
			"message":  "Missing-user synchronization finished",
			"inserted": inserted,
			"skipped":  skipped,
			"failed":   len(failures),
			"failures": failures,
			"source":   "postgresql",
			"target":   "yugabyte",
		})
	}
}

package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

func RepairUsersHandler(postgresDB, yugabyteDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"success": false,
				"message": "Method not allowed; use POST",
			})
			return
		}

		// حذف رکوردهای اضافه فقط با درخواست صریح فعال می‌شود.
		var request struct {
			DeleteExtra bool `json:"delete_extra"`
		}

		if r.Body != nil {
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()

			if err := decoder.Decode(&request); err != nil && err.Error() != "EOF" {
				writeJSON(w, http.StatusBadRequest, map[string]interface{}{
					"success": false,
					"message": "Invalid request body",
				})
				return
			}
		}

		if err := postgresDB.Ping(); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				"success": false,
				"message": "PostgreSQL is unavailable",
			})
			return
		}

		if err := yugabyteDB.Ping(); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				"success": false,
				"message": "YugabyteDB is unavailable",
			})
			return
		}

		source, err := loadUsers(postgresDB)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to read PostgreSQL users",
			})
			return
		}

		target, err := loadUsers(yugabyteDB)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to read YugabyteDB users",
			})
			return
		}

		emails := make([]string, 0, len(source))
		for email := range source {
			emails = append(emails, email)
		}
		sort.Strings(emails)

		inserted := 0
		updated := 0
		unchanged := 0
		deleted := 0
		failed := make([]map[string]string, 0)

		// مرحله اول: درج و اصلاح اطلاعات بر اساس PostgreSQL.
		for _, email := range emails {
			src := source[email]

			if strings.TrimSpace(src.FirstName) == "" ||
				strings.TrimSpace(src.LastName) == "" ||
				strings.TrimSpace(src.Email) == "" {
				failed = append(failed, map[string]string{
					"email":  email,
					"reason": "Source record has an empty required field",
				})
				continue
			}

			dst, exists := target[email]

			var age interface{}
			if src.Age != nil {
				age = *src.Age
			}

			if !exists {
				result, err := yugabyteDB.Exec(`
					INSERT INTO users (first_name, last_name, email, age)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (email) DO NOTHING
				`, src.FirstName, src.LastName, src.Email, age)

				if err != nil {
					failed = append(failed, map[string]string{
						"email":  email,
						"reason": "Failed to insert missing user",
					})
					continue
				}

				affected, err := result.RowsAffected()
				if err != nil {
					failed = append(failed, map[string]string{
						"email":  email,
						"reason": "Could not verify insert result",
					})
					continue
				}

				if affected == 1 {
					inserted++
				} else {
					unchanged++
				}
				continue
			}

			if src.FirstName == dst.FirstName &&
				src.LastName == dst.LastName &&
				sameAge(src.Age, dst.Age) {
				unchanged++
				continue
			}

			result, err := yugabyteDB.Exec(`
				UPDATE users
				SET first_name = $1,
					last_name = $2,
					age = $3,
					updated_at = CURRENT_TIMESTAMP
				WHERE email = $4
			`, src.FirstName, src.LastName, age, email)

			if err != nil {
				failed = append(failed, map[string]string{
					"email":  email,
					"reason": "Failed to update mismatched user",
				})
				continue
			}

			affected, err := result.RowsAffected()
			if err != nil || affected != 1 {
				failed = append(failed, map[string]string{
					"email":  email,
					"reason": "Could not confirm user update",
				})
				continue
			}

			updated++
		}

		// مرحله دوم: حذف رکوردهای اضافه، فقط در صورت درخواست صریح.
		if request.DeleteExtra {
			extraEmails := make([]string, 0)

			for email := range target {
				if _, exists := source[email]; !exists {
					extraEmails = append(extraEmails, email)
				}
			}
			sort.Strings(extraEmails)

			for _, email := range extraEmails {
				result, err := yugabyteDB.Exec(
					`DELETE FROM users WHERE email = $1`,
					email,
				)

				if err != nil {
					failed = append(failed, map[string]string{
						"email":  email,
						"reason": "Failed to delete extra user",
					})
					continue
				}

				affected, err := result.RowsAffected()
				if err != nil || affected != 1 {
					failed = append(failed, map[string]string{
						"email":  email,
						"reason": "Could not confirm extra user deletion",
					})
					continue
				}

				deleted++
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":              len(failed) == 0,
			"message":              "Database repair finished",
			"source":               "postgresql",
			"target":               "yugabyte",
			"inserted":             inserted,
			"updated":              updated,
			"unchanged":            unchanged,
			"deleted":              deleted,
			"delete_extra_enabled": request.DeleteExtra,
			"failed":               len(failed),
			"failures":             failed,
		})
	}
}

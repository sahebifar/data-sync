package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"data-sync/models"
)

func UsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT id, first_name, last_name, email, age, created_at, updated_at
			FROM users
			ORDER BY id
		`)
		if err != nil {
			http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		users := make([]models.User, 0)  // ← تغییر اصلی: به‌جای var users

		for rows.Next() {
			var user models.User
			err := rows.Scan(
				&user.ID,
				&user.FirstName,
				&user.LastName,
				&user.Email,
				&user.Age,
				&user.CreatedAt,
				&user.UpdatedAt,
			)
			if err != nil {
				http.Error(w, "Failed to read user", http.StatusInternalServerError)
				return
			}
			users = append(users, user)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}
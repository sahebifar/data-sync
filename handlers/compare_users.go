package handlers

import (
	"database/sql"
	"net/http"
)

type UserRecord struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Age       *int   `json:"age"`
}

type UserDifference struct {
	Email           string      `json:"email"`
	Status          string      `json:"status"`
	PostgresUser    *UserRecord `json:"postgres_user,omitempty"`
	YugabyteUser    *UserRecord `json:"yugabyte_user,omitempty"`
	DifferentFields []string    `json:"different_fields,omitempty"`
}

func CompareUsersHandler(postgresDB, yugabyteDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"success": false,
				"message": "Method not allowed",
			})
			return
		}

		postgresUsers, err := loadUsers(postgresDB)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to read PostgreSQL users",
			})
			return
		}

		yugabyteUsers, err := loadUsers(yugabyteDB)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to read YugabyteDB users",
			})
			return
		}

		differences := CompareUserMaps(postgresUsers, yugabyteUsers)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success":           true,
			"differences_count": len(differences),
			"differences":       differences,
		})
	}
}

func sameAge(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}

func loadUsers(db *sql.DB) (map[string]UserRecord, error) {
	rows, err := db.Query(`
		SELECT id, first_name, last_name, email, age
		FROM users
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[string]UserRecord)

	for rows.Next() {
		var user UserRecord
		var age sql.NullInt64

		if err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
			&age,
		); err != nil {
			return nil, err
		}

		if age.Valid {
			value := int(age.Int64)
			user.Age = &value
		}

		users[user.Email] = user
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

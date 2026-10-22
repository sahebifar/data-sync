package routes

import (
	"database/sql"
	"net/http"

	"data-sync/handlers"
)

func SetupRoutes(postgresDB, yugabyteDB *sql.DB) {
	http.Handle("/api/health", http.HandlerFunc(healthHandler))
	http.Handle("/api/users", handlers.UsersHandler(postgresDB))
	http.Handle("/api/users/yugabyte", handlers.UsersHandler(yugabyteDB))
	http.Handle("/api/users/create", handlers.CreateUserHandler(postgresDB, yugabyteDB))
	http.Handle("/api/users/update", handlers.UpdateUserHandler(postgresDB, yugabyteDB))
	http.Handle("/api/users/delete", handlers.DeleteUserHandler(postgresDB, yugabyteDB))

	http.Handle("/api/users/compare", handlers.CompareUsersHandler(postgresDB, yugabyteDB))
	http.Handle("/api/users/sync/missing", handlers.SyncMissingUsersHandler(postgresDB, yugabyteDB))
	http.Handle("/api/users/repair", handlers.RepairUsersHandler(postgresDB, yugabyteDB))
	http.Handle("/api/sync/outbox", handlers.OutboxListHandler(postgresDB))
	http.Handle("/api/sync/outbox/retry", handlers.RetryOutboxHandler(postgresDB))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("Method not allowed"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Data Sync Backend is running"))
}

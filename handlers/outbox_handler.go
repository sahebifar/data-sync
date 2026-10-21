package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
)

type outboxItem struct {
	ID          int64     `json:"id"`
	EntityType  string    `json:"entity_type"`
	EntityKey   string    `json:"entity_key"`
	Operation   string    `json:"operation"`
	Status      string    `json:"status"`
	Attempts    int       `json:"attempts"`
	NextRetryAt time.Time `json:"next_retry_at"`
	LastError   *string   `json:"last_error"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GET /api/sync/outbox
func OutboxListHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"success": false,
				"message": "Method not allowed; use GET",
			})
			return
		}

		status := r.URL.Query().Get("status")
		switch status {
		case "", "pending", "processing", "completed", "failed":
		default:
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "Invalid status filter",
			})
			return
		}

		rows, err := db.QueryContext(r.Context(), `
			SELECT id, entity_type, entity_key, operation, status,
			       attempts, next_retry_at, last_error, created_at, updated_at
			FROM sync_outbox
			WHERE ($1 = '' OR status = $1)
			ORDER BY id DESC
			LIMIT 100
		`, status)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to read outbox",
			})
			return
		}
		defer rows.Close()

		items := make([]outboxItem, 0)

		for rows.Next() {
			var item outboxItem
			var lastError sql.NullString

			err := rows.Scan(
				&item.ID,
				&item.EntityType,
				&item.EntityKey,
				&item.Operation,
				&item.Status,
				&item.Attempts,
				&item.NextRetryAt,
				&lastError,
				&item.CreatedAt,
				&item.UpdatedAt,
			)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
					"success": false,
					"message": "Failed to read outbox record",
				})
				return
			}

			if lastError.Valid {
				item.LastError = &lastError.String
			}

			items = append(items, item)
		}

		if err := rows.Err(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to iterate outbox records",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"count":   len(items),
			"data":    items,
		})
	})
}

// POST /api/sync/outbox/retry?id=123
func RetryOutboxHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"success": false,
				"message": "Method not allowed; use POST",
			})
			return
		}

		id, err := parsePositiveID(r.URL.Query().Get("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "A valid outbox id is required",
			})
			return
		}

		result, err := db.ExecContext(r.Context(), `
			UPDATE sync_outbox
			SET status = 'pending',
			    attempts = 0,
			    next_retry_at = CURRENT_TIMESTAMP,
			    last_error = NULL,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND status = 'failed'
		`, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Failed to queue outbox retry",
			})
			return
		}

		affected, err := result.RowsAffected()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Could not confirm outbox retry",
			})
			return
		}

		if affected == 0 {
			var exists bool
			err := db.QueryRowContext(r.Context(),
				`SELECT EXISTS (SELECT 1 FROM sync_outbox WHERE id = $1)`,
				id,
			).Scan(&exists)

			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
					"success": false,
					"message": "Failed to check outbox record",
				})
				return
			}

			if !exists {
				writeJSON(w, http.StatusNotFound, map[string]interface{}{
					"success": false,
					"message": "Outbox record not found",
				})
				return
			}

			writeJSON(w, http.StatusConflict, map[string]interface{}{
				"success": false,
				"message": "Only failed outbox records can be retried",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Outbox record queued for retry",
			"data": map[string]interface{}{
				"id":     id,
				"status": "pending",
			},
		})
	})
}

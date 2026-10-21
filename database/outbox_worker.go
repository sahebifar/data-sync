package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"time"
)

type syncPayload struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
}

// StartOutboxWorker صف همگام‌سازی را به‌صورت مداوم بررسی می‌کند.
func StartOutboxWorker(
	ctx context.Context,
	postgresDB *sql.DB,
	yugabyteDB *sql.DB,
) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Println("Outbox worker started")

	for {
		if ctx.Err() != nil {
			return
		}

		processed, err := processNextOutbox(ctx, postgresDB, yugabyteDB)
		if err != nil {
			log.Printf("Outbox worker error: %v", err)
		}

		if processed {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func processNextOutbox(
	ctx context.Context,
	postgresDB *sql.DB,
	yugabyteDB *sql.DB,
) (bool, error) {
	tx, err := postgresDB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var (
		id        int64
		entityKey string
		operation string
		payload   []byte
		attempts  int
	)

	err = tx.QueryRowContext(ctx, `
		SELECT id, entity_key, operation, payload, attempts
		FROM sync_outbox
		WHERE (
			status = 'pending'
			AND next_retry_at <= CURRENT_TIMESTAMP
		)
		OR (
			status = 'processing'
			AND updated_at < CURRENT_TIMESTAMP - INTERVAL '5 minutes'
		)
		ORDER BY id
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&id, &entityKey, &operation, &payload, &attempts)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	attempts++

	_, err = tx.ExecContext(ctx, `
		UPDATE sync_outbox
		SET status = 'processing',
		    attempts = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, attempts, id)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	var syncErr error

	if err := yugabyteDB.PingContext(ctx); err != nil {
		syncErr = err
	} else {
		syncErr = applySyncOperation(
			ctx,
			yugabyteDB,
			entityKey,
			operation,
			payload,
		)
	}

	if syncErr != nil {
		delaySeconds := 1 << min(attempts, 8)
		nextRetry := time.Now().Add(
			time.Duration(delaySeconds) * time.Second,
		)

		status := "pending"
		if attempts >= 10 {
			status = "failed"
		}

		_, updateErr := postgresDB.ExecContext(ctx, `
			UPDATE sync_outbox
			SET status = $1,
			    next_retry_at = $2,
			    last_error = $3,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $4
		`, status, nextRetry, syncErr.Error(), id)

		if updateErr != nil {
			return true, updateErr
		}

		log.Printf(
			"Outbox item %d failed (attempt %d); status=%s: %v",
			id, attempts, status, syncErr,
		)

		return true, nil
	}

	_, err = postgresDB.ExecContext(ctx, `
		UPDATE sync_outbox
		SET status = 'completed',
		    last_error = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id)
	if err != nil {
		return true, err
	}

	log.Printf("Outbox item %d synchronized successfully", id)
	return true, nil
}

func applySyncOperation(
	ctx context.Context,
	yugabyteDB *sql.DB,
	entityKey string,
	operation string,
	rawPayload []byte,
) error {
	switch operation {
	case "create", "update":
		var payload syncPayload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return err
		}
		if payload.Email == "" {
			payload.Email = entityKey
		}

		if payload.Email == "" {
			return errors.New("outbox payload has no email")
		}

		if payload.FirstName == "" {
			return errors.New("outbox payload has no first name")
		}

		if payload.LastName == "" {
			return errors.New("outbox payload has no last name")
		}

		if payload.Age <= 0 {
			return errors.New("outbox payload has invalid age")
		}

		if payload.Email == "" {
			payload.Email = entityKey
		}

		_, err := yugabyteDB.ExecContext(ctx, `
			INSERT INTO users (
				first_name, last_name, email, age, updated_at
			)
			VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
			ON CONFLICT (email)
			DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				age = EXCLUDED.age,
				updated_at = CURRENT_TIMESTAMP
		`,
			payload.FirstName,
			payload.LastName,
			payload.Email,
			payload.Age,
		)
		return err

	case "delete":
		_, err := yugabyteDB.ExecContext(ctx, `
			DELETE FROM users
			WHERE LOWER(email) = LOWER($1)
		`, entityKey)
		return err

	default:
		return errors.New("unsupported outbox operation: " + operation)
	}
}

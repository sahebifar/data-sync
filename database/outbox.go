package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// EnqueueSync با استفاده از تراکنش PostgreSQL،
// درخواست همگام‌سازی را در صف ذخیره می‌کند.
func EnqueueSync(
	tx *sql.Tx,
	entityKey string,
	operation string,
	payload interface{},
) error {
	if operation != "create" &&
		operation != "update" &&
		operation != "delete" {
		return fmt.Errorf("invalid sync operation: %s", operation)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode sync payload: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO sync_outbox (
			entity_type,
			entity_key,
			operation,
			payload,
			status
		)
		VALUES ('user', $1, $2, $3::jsonb, 'pending')
	`,
		entityKey,
		operation,
		string(payloadJSON),
	)
	if err != nil {
		return fmt.Errorf("insert sync outbox record: %w", err)
	}

	return nil
}

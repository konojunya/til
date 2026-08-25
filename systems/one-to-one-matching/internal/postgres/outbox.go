package postgres

import (
	"context"
	"encoding/json"
	"fmt"
)

type OutboxEvent struct {
	ID        string
	EventType string
	Payload   json.RawMessage
}

func (r *Repository) SaveOutboxEvent(ctx context.Context, event OutboxEvent) error {
	_, err := r.db.Exec(
		ctx,
		`
		INSERT INTO outbox_events (id, event_type, payload)
		VALUES ($1, $2, $3::jsonb)
		`,
		event.ID,
		event.EventType,
		event.Payload,
	)
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}

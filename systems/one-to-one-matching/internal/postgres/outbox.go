package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	sqldb "github.com/konojunya/til/systems/one-to-one-matching/internal/postgres/db"
)

type OutboxEvent struct {
	ID        string
	EventType string
	Payload   json.RawMessage
}

func (r *Repository) SaveOutboxEvent(ctx context.Context, event OutboxEvent) error {
	err := r.queries.SaveOutboxEvent(ctx, sqldb.SaveOutboxEventParams{
		ID:        event.ID,
		EventType: event.EventType,
		Payload:   event.Payload,
	})
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}

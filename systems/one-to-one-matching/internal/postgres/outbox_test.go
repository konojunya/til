//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"testing"
)

func TestOutboxSchemaStoresPendingEvent(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	const eventID = "outbox-event-1"

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO outbox_events (id, event_type, payload)
			VALUES ($1, $2, $3::jsonb)
		`,
		eventID,
		"match.created",
		`{
			"user_low_id": "outbox-alice",
			"user_high_id": "outbox-bob"
		}`,
	)
	if err != nil {
		t.Fatalf("failed to insert outbox event: %v", err)
	}

	var (
		gotEventType string
		gotLowID     string
		gotHighID    string
		gotPending   bool
	)

	err = tx.QueryRow(
		ctx,
		`
			SELECT
				event_type,
				payload ->> 'user_low_id',
				payload ->> 'user_high_id',
				published_at IS NULL
			FROM outbox_events
			WHERE id = $1
		`,
		eventID,
	).Scan(
		&gotEventType,
		&gotLowID,
		&gotHighID,
		&gotPending,
	)
	if err != nil {
		t.Fatalf("failed to read outbox event: %v", err)
	}

	if gotEventType != "match.created" {
		t.Fatalf("event type = %q, want %q", gotEventType, "match.created")
	}
	if gotLowID != "outbox-alice" {
		t.Fatalf("low user ID = %q, want %q", gotLowID, "outbox-alice")
	}
	if gotHighID != "outbox-bob" {
		t.Fatalf("high user ID = %q, want %q", gotHighID, "outbox-bob")
	}
	if !gotPending {
		t.Fatal("expected outbox event to be pending")
	}
}

func TestRepositorySaveOutboxEvent(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	event := OutboxEvent{
		ID:        "repository-outbox-event-1",
		EventType: "match.created",
		Payload: json.RawMessage(`{
			"user_low_id": "repository-outbox-alice",
			"user_high_id": "repository-outbox-bob"
		}`),
	}

	repo := NewRepository(tx)

	if err := repo.SaveOutboxEvent(ctx, event); err != nil {
		t.Fatalf("failed to save outbox event: %v", err)
	}

	var (
		gotEventType string
		gotLowID     string
		gotHighID    string
		gotPending   bool
	)

	err = tx.QueryRow(
		ctx,
		`
			SELECT
				event_type,
				payload ->> 'user_low_id',
				payload ->> 'user_high_id',
				published_at IS NULL
			FROM outbox_events
			WHERE id = $1
		`,
		event.ID,
	).Scan(
		&gotEventType,
		&gotLowID,
		&gotHighID,
		&gotPending,
	)
	if err != nil {
		t.Fatalf("failed to read outbox event: %v", err)
	}

	if gotEventType != event.EventType {
		t.Fatalf("event type = %q, want %q", gotEventType, event.EventType)
	}
	if gotLowID != "repository-outbox-alice" {
		t.Fatalf("low user ID = %q, want %q", gotLowID, "repository-outbox-alice")
	}
	if gotHighID != "repository-outbox-bob" {
		t.Fatalf("high user ID = %q, want %q", gotHighID, "repository-outbox-bob")
	}
	if !gotPending {
		t.Fatal("expected outbox event to be pending")
	}
}

//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestTransactorCommitsOnSuccess(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const eventID = "transaction-commit-event"

	if _, err := conn.Exec(
		ctx,
		`DELETE FROM outbox_events WHERE id = $1`,
		eventID,
	); err != nil {
		t.Fatalf("failed to clean event before test: %v", err)
	}

	t.Cleanup(func() {
		if _, err := conn.Exec(
			context.Background(),
			`DELETE FROM outbox_events WHERE id = $1`,
			eventID,
		); err != nil {
			t.Errorf("failed to clean committed event: %v", err)
		}
	})

	transactor := NewTransactor(conn)

	err := transactor.WithinTransaction(ctx, func(repo *Repository) error {
		return repo.SaveOutboxEvent(ctx, OutboxEvent{
			ID:        eventID,
			EventType: "match.created",
			Payload: json.RawMessage(`{
				"user_low_id": "transaction-commit-alice",
				"user_high_id": "transaction-commit-bob"
			}`),
		})
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	var got int
	if err := conn.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM outbox_events WHERE id = $1`,
		eventID,
	).Scan(&got); err != nil {
		t.Fatalf("failed to count committed event: %v", err)
	}

	if got != 1 {
		t.Fatalf("committed event count = %d, want 1", got)
	}
}

func TestTransactorRollsBackOnCallbackError(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const eventID = "transaction-rollback-event"

	if _, err := conn.Exec(
		ctx,
		`DELETE FROM outbox_events WHERE id = $1`,
		eventID,
	); err != nil {
		t.Fatalf("failed to clean event before test: %v", err)
	}

	forcedErr := errors.New("force rollback")
	transactor := NewTransactor(conn)

	err := transactor.WithinTransaction(ctx, func(repo *Repository) error {
		if err := repo.SaveOutboxEvent(ctx, OutboxEvent{
			ID:        eventID,
			EventType: "match.created",
			Payload: json.RawMessage(`{
				"user_low_id": "transaction-rollback-alice",
				"user_high_id": "transaction-rollback-bob"
			}`),
		}); err != nil {
			return err
		}

		return forcedErr
	})
	if !errors.Is(err, forcedErr) {
		t.Fatalf("transaction error = %v, want %v", err, forcedErr)
	}

	var got int
	if err := conn.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM outbox_events WHERE id = $1`,
		eventID,
	).Scan(&got); err != nil {
		t.Fatalf("failed to count rolled back event: %v", err)
	}

	if got != 0 {
		t.Fatalf("rolled back event count = %d, want 0", got)
	}
}

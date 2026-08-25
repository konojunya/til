//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

func TestMatchingServiceCommitsOneWayLikeWithoutMatchOrOutbox(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue = "application-one-way-alice"
		bobValue   = "application-one-way-bob"
	)

	if err := deleteMatchingServiceFixture(
		ctx,
		conn,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to clean fixture before test: %v", err)
	}

	t.Cleanup(func() {
		if err := deleteMatchingServiceFixture(
			context.Background(),
			conn,
			aliceValue,
			bobValue,
		); err != nil {
			t.Errorf("failed to clean fixture after test: %v", err)
		}
	})

	if _, err := conn.Exec(
		ctx,
		`INSERT INTO users (id) VALUES ($1), ($2)`,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	alice, err := matching.NewUserID(aliceValue)
	if err != nil {
		t.Fatalf("failed to create Alice UserID: %v", err)
	}

	bob, err := matching.NewUserID(bobValue)
	if err != nil {
		t.Fatalf("failed to create Bob UserID: %v", err)
	}

	var outboxBefore int
	if err := conn.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM outbox_events`,
	).Scan(&outboxBefore); err != nil {
		t.Fatalf("failed to count outbox before SendLike: %v", err)
	}

	service := NewMatchingService(
		NewTransactor(conn),
		func() string {
			t.Fatal("event ID generator must not be called without a match")
			return ""
		},
	)

	result, err := service.SendLike(ctx, alice, bob)
	if err != nil {
		t.Fatalf("SendLike() error = %v", err)
	}

	if !result.LikeCreated {
		t.Fatal("SendLike() LikeCreated = false, want true")
	}

	if result.MatchCreated {
		t.Fatal("SendLike() MatchCreated = true, want false")
	}

	var likeCount int
	if err := conn.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM likes
			WHERE sender_id = $1
			  AND receiver_id = $2
		`,
		aliceValue,
		bobValue,
	).Scan(&likeCount); err != nil {
		t.Fatalf("failed to count likes: %v", err)
	}

	if likeCount != 1 {
		t.Fatalf("like count = %d, want 1", likeCount)
	}

	var matchCount int
	if err := conn.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM matches
			WHERE user_low_id IN ($1, $2)
			  AND user_high_id IN ($1, $2)
		`,
		aliceValue,
		bobValue,
	).Scan(&matchCount); err != nil {
		t.Fatalf("failed to count matches: %v", err)
	}

	if matchCount != 0 {
		t.Fatalf("match count = %d, want 0", matchCount)
	}

	var outboxAfter int
	if err := conn.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM outbox_events`,
	).Scan(&outboxAfter); err != nil {
		t.Fatalf("failed to count outbox after SendLike: %v", err)
	}

	if outboxAfter != outboxBefore {
		t.Fatalf(
			"outbox count = %d, want unchanged count %d",
			outboxAfter,
			outboxBefore,
		)
	}
}

func deleteMatchingServiceFixture(
	ctx context.Context,
	conn *pgx.Conn,
	firstUserID string,
	secondUserID string,
) error {
	steps := []struct {
		name  string
		query string
	}{
		{
			name: "matches",
			query: `
				DELETE FROM matches
				WHERE user_low_id IN ($1, $2)
				   OR user_high_id IN ($1, $2)
			`,
		},
		{
			name: "likes",
			query: `
				DELETE FROM likes
				WHERE sender_id IN ($1, $2)
				   OR receiver_id IN ($1, $2)
			`,
		},
		{
			name:  "users",
			query: `DELETE FROM users WHERE id IN ($1, $2)`,
		},
	}

	for _, step := range steps {
		if _, err := conn.Exec(
			ctx,
			step.query,
			firstUserID,
			secondUserID,
		); err != nil {
			return fmt.Errorf("delete %s: %w", step.name, err)
		}
	}

	return nil
}

func TestMatchingServiceCommitsMatchAndOutboxForMutualLikes(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue = "application-mutual-alice"
		bobValue   = "application-mutual-bob"
		eventID    = "application-mutual-match-created"
	)

	if err := deleteMatchingServiceFixture(
		ctx,
		conn,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to clean fixture before test: %v", err)
	}

	if _, err := conn.Exec(
		ctx,
		`DELETE FROM outbox_events WHERE id = $1`,
		eventID,
	); err != nil {
		t.Fatalf("failed to clean outbox before test: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()

		if _, err := conn.Exec(
			cleanupCtx,
			`DELETE FROM outbox_events WHERE id = $1`,
			eventID,
		); err != nil {
			t.Errorf("failed to clean outbox after test: %v", err)
		}

		if err := deleteMatchingServiceFixture(
			cleanupCtx,
			conn,
			aliceValue,
			bobValue,
		); err != nil {
			t.Errorf("failed to clean fixture after test: %v", err)
		}
	})

	if _, err := conn.Exec(
		ctx,
		`INSERT INTO users (id) VALUES ($1), ($2)`,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	alice, err := matching.NewUserID(aliceValue)
	if err != nil {
		t.Fatalf("failed to create Alice UserID: %v", err)
	}

	bob, err := matching.NewUserID(bobValue)
	if err != nil {
		t.Fatalf("failed to create Bob UserID: %v", err)
	}

	eventIDCalls := 0

	service := NewMatchingService(
		NewTransactor(conn),
		func() string {
			eventIDCalls++
			return eventID
		},
	)

	first, err := service.SendLike(ctx, alice, bob)
	if err != nil {
		t.Fatalf("SendLike(alice, bob) error = %v", err)
	}

	if !first.LikeCreated {
		t.Fatal("first SendLike() LikeCreated = false, want true")
	}

	if first.MatchCreated {
		t.Fatal("first SendLike() MatchCreated = true, want false")
	}

	second, err := service.SendLike(ctx, bob, alice)
	if err != nil {
		t.Fatalf("SendLike(bob, alice) error = %v", err)
	}

	if !second.LikeCreated {
		t.Fatal("second SendLike() LikeCreated = false, want true")
	}

	if !second.MatchCreated {
		t.Fatal("second SendLike() MatchCreated = false, want true")
	}

	var likeCount int
	if err := conn.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM likes
			WHERE (sender_id = $1 AND receiver_id = $2)
			   OR (sender_id = $2 AND receiver_id = $1)
		`,
		aliceValue,
		bobValue,
	).Scan(&likeCount); err != nil {
		t.Fatalf("failed to count likes: %v", err)
	}

	if likeCount != 2 {
		t.Fatalf("like count = %d, want 2", likeCount)
	}

	var matchCount int
	if err := conn.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM matches
			WHERE user_low_id = $1
			  AND user_high_id = $2
		`,
		aliceValue,
		bobValue,
	).Scan(&matchCount); err != nil {
		t.Fatalf("failed to count matches: %v", err)
	}

	if matchCount != 1 {
		t.Fatalf("match count = %d, want 1", matchCount)
	}

	if eventIDCalls != 1 {
		t.Fatalf(
			"event ID generator calls = %d, want 1",
			eventIDCalls,
		)
	}

	var (
		gotEventType string
		gotLowID     string
		gotHighID    string
		gotPending   bool
	)

	if err := conn.QueryRow(
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
	); err != nil {
		t.Fatalf("failed to read outbox event: %v", err)
	}

	if gotEventType != "match.created" {
		t.Fatalf(
			"event type = %q, want %q",
			gotEventType,
			"match.created",
		)
	}

	if gotLowID != aliceValue {
		t.Fatalf("low user ID = %q, want %q", gotLowID, aliceValue)
	}

	if gotHighID != bobValue {
		t.Fatalf("high user ID = %q, want %q", gotHighID, bobValue)
	}

	if !gotPending {
		t.Fatal("outbox event is already published, want pending")
	}
}

func TestMatchingServiceDoesNotDuplicateStateWhenLikeIsResent(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue = "application-idempotent-alice"
		bobValue   = "application-idempotent-bob"
		eventID    = "application-idempotent-match-created"
	)

	if err := deleteMatchingServiceFixture(
		ctx,
		conn,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to clean fixture before test: %v", err)
	}

	if _, err := conn.Exec(
		ctx,
		`DELETE FROM outbox_events WHERE id = $1`,
		eventID,
	); err != nil {
		t.Fatalf("failed to clean outbox before test: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()

		if _, err := conn.Exec(
			cleanupCtx,
			`DELETE FROM outbox_events WHERE id = $1`,
			eventID,
		); err != nil {
			t.Errorf("failed to clean outbox after test: %v", err)
		}

		if err := deleteMatchingServiceFixture(
			cleanupCtx,
			conn,
			aliceValue,
			bobValue,
		); err != nil {
			t.Errorf("failed to clean fixture after test: %v", err)
		}
	})

	if _, err := conn.Exec(
		ctx,
		`INSERT INTO users (id) VALUES ($1), ($2)`,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	alice, err := matching.NewUserID(aliceValue)
	if err != nil {
		t.Fatalf("failed to create Alice UserID: %v", err)
	}

	bob, err := matching.NewUserID(bobValue)
	if err != nil {
		t.Fatalf("failed to create Bob UserID: %v", err)
	}

	eventIDCalls := 0

	service := NewMatchingService(
		NewTransactor(conn),
		func() string {
			eventIDCalls++
			return eventID
		},
	)

	if _, err := service.SendLike(ctx, alice, bob); err != nil {
		t.Fatalf("SendLike(alice, bob) error = %v", err)
	}

	matched, err := service.SendLike(ctx, bob, alice)
	if err != nil {
		t.Fatalf("SendLike(bob, alice) error = %v", err)
	}

	if !matched.MatchCreated {
		t.Fatal("setup SendLike() MatchCreated = false, want true")
	}

	resends := []struct {
		name     string
		sender   matching.UserID
		receiver matching.UserID
	}{
		{
			name:     "resend alice to bob",
			sender:   alice,
			receiver: bob,
		},
		{
			name:     "resend bob to alice",
			sender:   bob,
			receiver: alice,
		},
	}

	for _, resend := range resends {
		t.Run(resend.name, func(t *testing.T) {
			result, err := service.SendLike(
				ctx,
				resend.sender,
				resend.receiver,
			)
			if err != nil {
				t.Fatalf("SendLike() error = %v", err)
			}

			if result.LikeCreated {
				t.Fatal("SendLike() LikeCreated = true, want false")
			}

			if result.MatchCreated {
				t.Fatal("SendLike() MatchCreated = true, want false")
			}
		})
	}

	if eventIDCalls != 1 {
		t.Fatalf(
			"event ID generator calls = %d, want 1",
			eventIDCalls,
		)
	}

	var (
		likeCount   int
		matchCount  int
		outboxCount int
	)

	if err := conn.QueryRow(
		ctx,
		`
			SELECT
				(
					SELECT COUNT(*)
					FROM likes
					WHERE (sender_id = $1 AND receiver_id = $2)
					   OR (sender_id = $2 AND receiver_id = $1)
				),
				(
					SELECT COUNT(*)
					FROM matches
					WHERE user_low_id = $1
					  AND user_high_id = $2
				),
				(
					SELECT COUNT(*)
					FROM outbox_events
					WHERE id = $3
				)
		`,
		aliceValue,
		bobValue,
		eventID,
	).Scan(
		&likeCount,
		&matchCount,
		&outboxCount,
	); err != nil {
		t.Fatalf("failed to count persisted state: %v", err)
	}

	if likeCount != 2 {
		t.Fatalf("like count = %d, want 2", likeCount)
	}

	if matchCount != 1 {
		t.Fatalf("match count = %d, want 1", matchCount)
	}

	if outboxCount != 1 {
		t.Fatalf("outbox count = %d, want 1", outboxCount)
	}
}

func TestMatchingServiceRollsBackMatchWhenOutboxSaveFails(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue = "application-rollback-alice"
		bobValue   = "application-rollback-bob"
	)

	if _, err := conn.Exec(
		ctx,
		`
			DELETE FROM outbox_events
			WHERE payload ->> 'user_low_id' = $1
			  AND payload ->> 'user_high_id' = $2
		`,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to clean outbox before test: %v", err)
	}

	if err := deleteMatchingServiceFixture(
		ctx,
		conn,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to clean fixture before test: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()

		if _, err := conn.Exec(
			cleanupCtx,
			`
				DELETE FROM outbox_events
				WHERE payload ->> 'user_low_id' = $1
				  AND payload ->> 'user_high_id' = $2
			`,
			aliceValue,
			bobValue,
		); err != nil {
			t.Errorf("failed to clean outbox after test: %v", err)
		}

		if err := deleteMatchingServiceFixture(
			cleanupCtx,
			conn,
			aliceValue,
			bobValue,
		); err != nil {
			t.Errorf("failed to clean fixture after test: %v", err)
		}
	})

	if _, err := conn.Exec(
		ctx,
		`INSERT INTO users (id) VALUES ($1), ($2)`,
		aliceValue,
		bobValue,
	); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	alice, err := matching.NewUserID(aliceValue)
	if err != nil {
		t.Fatalf("failed to create Alice UserID: %v", err)
	}

	bob, err := matching.NewUserID(bobValue)
	if err != nil {
		t.Fatalf("failed to create Bob UserID: %v", err)
	}

	eventIDCalls := 0

	service := NewMatchingService(
		NewTransactor(conn),
		func() string {
			eventIDCalls++

			// outbox_events_id_not_emptyを意図的に違反させる。
			return ""
		},
	)

	first, err := service.SendLike(ctx, alice, bob)
	if err != nil {
		t.Fatalf("SendLike(alice, bob) error = %v", err)
	}

	if !first.LikeCreated {
		t.Fatal("first SendLike() LikeCreated = false, want true")
	}

	failedResult, err := service.SendLike(ctx, bob, alice)
	if err == nil {
		t.Fatal("SendLike(bob, alice) error = nil, want error")
	}

	if failedResult.LikeCreated {
		t.Fatal("failed SendLike() LikeCreated = true, want false")
	}

	if failedResult.MatchCreated {
		t.Fatal("failed SendLike() MatchCreated = true, want false")
	}

	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		t.Fatalf(
			"SendLike() error type = %T, want *pgconn.PgError in chain",
			err,
		)
	}

	if postgresError.Code != "23514" {
		t.Fatalf(
			"PostgreSQL error code = %q, want %q",
			postgresError.Code,
			"23514",
		)
	}

	if postgresError.ConstraintName != "outbox_events_id_not_empty" {
		t.Fatalf(
			"constraint = %q, want %q",
			postgresError.ConstraintName,
			"outbox_events_id_not_empty",
		)
	}

	if eventIDCalls != 1 {
		t.Fatalf(
			"event ID generator calls = %d, want 1",
			eventIDCalls,
		)
	}

	var (
		forwardLikeCount int
		reverseLikeCount int
		matchCount       int
		outboxCount      int
	)

	if err := conn.QueryRow(
		ctx,
		`
			SELECT
				(
					SELECT COUNT(*)
					FROM likes
					WHERE sender_id = $1
					  AND receiver_id = $2
				),
				(
					SELECT COUNT(*)
					FROM likes
					WHERE sender_id = $2
					  AND receiver_id = $1
				),
				(
					SELECT COUNT(*)
					FROM matches
					WHERE user_low_id = $1
					  AND user_high_id = $2
				),
				(
					SELECT COUNT(*)
					FROM outbox_events
					WHERE payload ->> 'user_low_id' = $1
					  AND payload ->> 'user_high_id' = $2
				)
		`,
		aliceValue,
		bobValue,
	).Scan(
		&forwardLikeCount,
		&reverseLikeCount,
		&matchCount,
		&outboxCount,
	); err != nil {
		t.Fatalf("failed to count persisted state: %v", err)
	}

	if forwardLikeCount != 1 {
		t.Fatalf(
			"committed forward like count = %d, want 1",
			forwardLikeCount,
		)
	}

	if reverseLikeCount != 0 {
		t.Fatalf(
			"rolled back reverse like count = %d, want 0",
			reverseLikeCount,
		)
	}

	if matchCount != 0 {
		t.Fatalf("rolled back match count = %d, want 0", matchCount)
	}

	if outboxCount != 0 {
		t.Fatalf("rolled back outbox count = %d, want 0", outboxCount)
	}
}

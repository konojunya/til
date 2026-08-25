//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

type hasLikeBarrierTracer struct {
	reached atomic.Int32
	release chan struct{}
	once    sync.Once
	maxWait time.Duration
}

func newHasLikeBarrierTracer(maxWait time.Duration) *hasLikeBarrierTracer {
	return &hasLikeBarrierTracer{
		release: make(chan struct{}),
		maxWait: maxWait,
	}
}

func (t *hasLikeBarrierTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	if !strings.Contains(data.SQL, "-- name: HasLike :one") {
		return ctx
	}

	if t.reached.Add(1) == 2 {
		t.releaseAll()
	}

	timer := time.NewTimer(t.maxWait)
	defer timer.Stop()

	select {
	case <-t.release:
	case <-timer.C:
		t.releaseAll()
	case <-ctx.Done():
		t.releaseAll()
	}

	return ctx
}

func (t *hasLikeBarrierTracer) TraceQueryEnd(
	context.Context,
	*pgx.Conn,
	pgx.TraceQueryEndData,
) {
}

func (t *hasLikeBarrierTracer) releaseAll() {
	t.once.Do(func() {
		close(t.release)
	})
}

func TestMatchingServiceConcurrentMutualLikesCreateOneMatchAndOutbox(
	t *testing.T,
) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue = "concurrency-alice"
		bobValue   = "concurrency-bob"
	)

	cleanup := func(ctx context.Context) error {
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
			return fmt.Errorf("delete outbox events: %w", err)
		}

		return deleteMatchingServiceFixture(
			ctx,
			conn,
			aliceValue,
			bobValue,
		)
	}

	if err := cleanup(ctx); err != nil {
		t.Fatalf("failed to clean fixture before test: %v", err)
	}

	t.Cleanup(func() {
		if err := cleanup(context.Background()); err != nil {
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

	tracer := newHasLikeBarrierTracer(time.Second)

	poolConfig, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("pgxpool.ParseConfig() error = %v", err)
	}

	poolConfig.MaxConns = 2
	poolConfig.ConnConfig.Tracer = tracer

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("pgxpool.NewWithConfig() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool.Ping() error = %v", err)
	}

	var eventIDCalls atomic.Int32

	service := NewMatchingService(
		NewTransactor(pool),
		func() string {
			call := eventIDCalls.Add(1)
			return fmt.Sprintf("concurrency-match-created-%d", call)
		},
	)

	type outcome struct {
		result matching.SendLikeResult
		err    error
	}

	start := make(chan struct{})
	outcomes := make(chan outcome, 2)

	sendLike := func(sender, receiver matching.UserID) {
		<-start

		result, err := service.SendLike(ctx, sender, receiver)
		outcomes <- outcome{
			result: result,
			err:    err,
		}
	}

	go sendLike(alice, bob)
	go sendLike(bob, alice)

	close(start)

	got := []outcome{
		<-outcomes,
		<-outcomes,
	}

	matchCreatedResults := 0

	for i, outcome := range got {
		if outcome.err != nil {
			t.Fatalf("SendLike()[%d] error = %v", i, outcome.err)
		}

		if !outcome.result.LikeCreated {
			t.Errorf("SendLike()[%d] LikeCreated = false, want true", i)
		}

		if outcome.result.MatchCreated {
			matchCreatedResults++
		}
	}

	if tracer.reached.Load() != 2 {
		t.Fatalf(
			"HasLike query count = %d, want 2",
			tracer.reached.Load(),
		)
	}

	if matchCreatedResults != 1 {
		t.Errorf(
			"MatchCreated result count = %d, want 1",
			matchCreatedResults,
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
					WHERE event_type = 'match.created'
					  AND payload ->> 'user_low_id' = $1
					  AND payload ->> 'user_high_id' = $2
				)
		`,
		aliceValue,
		bobValue,
	).Scan(
		&likeCount,
		&matchCount,
		&outboxCount,
	); err != nil {
		t.Fatalf("failed to count final state: %v", err)
	}

	if likeCount != 2 {
		t.Errorf("like count = %d, want 2", likeCount)
	}

	if matchCount != 1 {
		t.Errorf("match count = %d, want 1", matchCount)
	}

	if outboxCount != 1 {
		t.Errorf("outbox count = %d, want 1", outboxCount)
	}

	if eventIDCalls.Load() != 1 {
		t.Errorf(
			"event ID generator calls = %d, want 1",
			eventIDCalls.Load(),
		)
	}
}

func TestMatchingServiceConcurrentDuplicateLikesStayIdempotent(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue = "concurrency-duplicate-alice"
		bobValue   = "concurrency-duplicate-bob"
		workers    = 32
	)

	cleanup := func(ctx context.Context) error {
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
			return fmt.Errorf("delete outbox events: %w", err)
		}

		return deleteMatchingServiceFixture(
			ctx,
			conn,
			aliceValue,
			bobValue,
		)
	}

	if err := cleanup(ctx); err != nil {
		t.Fatalf("failed to clean fixture before test: %v", err)
	}

	t.Cleanup(func() {
		if err := cleanup(context.Background()); err != nil {
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

	poolConfig, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("pgxpool.ParseConfig() error = %v", err)
	}

	poolConfig.MaxConns = 8

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("pgxpool.NewWithConfig() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool.Ping() error = %v", err)
	}

	var eventIDCalls atomic.Int32

	service := NewMatchingService(
		NewTransactor(pool),
		func() string {
			call := eventIDCalls.Add(1)
			return fmt.Sprintf("unexpected-duplicate-event-%d", call)
		},
	)

	type outcome struct {
		result matching.SendLikeResult
		err    error
	}

	start := make(chan struct{})
	outcomes := make(chan outcome, workers)

	for range workers {
		go func() {
			<-start

			result, err := service.SendLike(ctx, alice, bob)
			outcomes <- outcome{
				result: result,
				err:    err,
			}
		}()
	}

	close(start)

	likeCreatedResults := 0
	matchCreatedResults := 0

	for i := range workers {
		outcome := <-outcomes
		if outcome.err != nil {
			t.Fatalf("SendLike()[%d] error = %v", i, outcome.err)
		}

		if outcome.result.LikeCreated {
			likeCreatedResults++
		}

		if outcome.result.MatchCreated {
			matchCreatedResults++
		}
	}

	if likeCreatedResults != 1 {
		t.Errorf(
			"LikeCreated result count = %d, want 1",
			likeCreatedResults,
		)
	}

	if matchCreatedResults != 0 {
		t.Errorf(
			"MatchCreated result count = %d, want 0",
			matchCreatedResults,
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
					WHERE sender_id = $1
					  AND receiver_id = $2
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
					WHERE event_type = 'match.created'
					  AND payload ->> 'user_low_id' = $1
					  AND payload ->> 'user_high_id' = $2
				)
		`,
		aliceValue,
		bobValue,
	).Scan(
		&likeCount,
		&matchCount,
		&outboxCount,
	); err != nil {
		t.Fatalf("failed to count final state: %v", err)
	}

	if likeCount != 1 {
		t.Errorf("like count = %d, want 1", likeCount)
	}

	if matchCount != 0 {
		t.Errorf("match count = %d, want 0", matchCount)
	}

	if outboxCount != 0 {
		t.Errorf("outbox count = %d, want 0", outboxCount)
	}

	if eventIDCalls.Load() != 0 {
		t.Errorf(
			"event ID generator calls = %d, want 0",
			eventIDCalls.Load(),
		)
	}
}

func TestMatchingServiceConcurrentMixedLikesConvergeToOneMatchAndOutbox(
	t *testing.T,
) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue          = "concurrency-mixed-alice"
		bobValue            = "concurrency-mixed-bob"
		workersPerDirection = 16
		totalWorkers        = workersPerDirection * 2
	)

	cleanup := func(ctx context.Context) error {
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
			return fmt.Errorf("delete outbox events: %w", err)
		}

		return deleteMatchingServiceFixture(
			ctx,
			conn,
			aliceValue,
			bobValue,
		)
	}

	if err := cleanup(ctx); err != nil {
		t.Fatalf("failed to clean fixture before test: %v", err)
	}

	t.Cleanup(func() {
		if err := cleanup(context.Background()); err != nil {
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

	poolConfig, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("pgxpool.ParseConfig() error = %v", err)
	}

	poolConfig.MaxConns = 8

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("pgxpool.NewWithConfig() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool.Ping() error = %v", err)
	}

	var eventIDCalls atomic.Int32

	service := NewMatchingService(
		NewTransactor(pool),
		func() string {
			call := eventIDCalls.Add(1)
			return fmt.Sprintf("concurrency-mixed-match-created-%d", call)
		},
	)

	type outcome struct {
		result matching.SendLikeResult
		err    error
	}

	start := make(chan struct{})
	outcomes := make(chan outcome, totalWorkers)

	sendLike := func(sender, receiver matching.UserID) {
		<-start

		result, err := service.SendLike(ctx, sender, receiver)
		outcomes <- outcome{
			result: result,
			err:    err,
		}
	}

	for range workersPerDirection {
		go sendLike(alice, bob)
		go sendLike(bob, alice)
	}

	close(start)

	likeCreatedResults := 0
	matchCreatedResults := 0

	for i := range totalWorkers {
		outcome := <-outcomes
		if outcome.err != nil {
			t.Fatalf("SendLike()[%d] error = %v", i, outcome.err)
		}

		if outcome.result.LikeCreated {
			likeCreatedResults++
		}

		if outcome.result.MatchCreated {
			matchCreatedResults++
		}
	}

	if likeCreatedResults != 2 {
		t.Errorf(
			"LikeCreated result count = %d, want 2",
			likeCreatedResults,
		)
	}

	if matchCreatedResults != 1 {
		t.Errorf(
			"MatchCreated result count = %d, want 1",
			matchCreatedResults,
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
					WHERE event_type = 'match.created'
					  AND payload ->> 'user_low_id' = $1
					  AND payload ->> 'user_high_id' = $2
				)
		`,
		aliceValue,
		bobValue,
	).Scan(
		&likeCount,
		&matchCount,
		&outboxCount,
	); err != nil {
		t.Fatalf("failed to count final state: %v", err)
	}

	if likeCount != 2 {
		t.Errorf("like count = %d, want 2", likeCount)
	}

	if matchCount != 1 {
		t.Errorf("match count = %d, want 1", matchCount)
	}

	if outboxCount != 1 {
		t.Errorf("outbox count = %d, want 1", outboxCount)
	}

	if eventIDCalls.Load() != 1 {
		t.Errorf(
			"event ID generator calls = %d, want 1",
			eventIDCalls.Load(),
		)
	}
}

func TestMatchingServiceLockWaitHonorsContextCancellation(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	const (
		aliceValue = "concurrency-cancel-alice"
		bobValue   = "concurrency-cancel-bob"
	)

	cleanup := func(ctx context.Context) error {
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
			return fmt.Errorf("delete outbox events: %w", err)
		}

		return deleteMatchingServiceFixture(
			ctx,
			conn,
			aliceValue,
			bobValue,
		)
	}

	if err := cleanup(ctx); err != nil {
		t.Fatalf("failed to clean fixture before test: %v", err)
	}

	t.Cleanup(func() {
		if err := cleanup(context.Background()); err != nil {
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

	lockTx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin lock holder transaction: %v", err)
	}
	t.Cleanup(func() {
		_ = lockTx.Rollback(context.Background())
	})

	if err := NewRepository(lockTx).LockMatchingPair(
		ctx,
		alice,
		bob,
	); err != nil {
		t.Fatalf("failed to hold matching pair lock: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("pgxpool.ParseConfig() error = %v", err)
	}

	poolConfig.MaxConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("pgxpool.NewWithConfig() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool.Ping() error = %v", err)
	}

	var eventIDCalls atomic.Int32

	service := NewMatchingService(
		NewTransactor(pool),
		func() string {
			call := eventIDCalls.Add(1)
			return fmt.Sprintf("unexpected-cancel-event-%d", call)
		},
	)

	requestCtx, cancel := context.WithTimeout(
		context.Background(),
		100*time.Millisecond,
	)
	defer cancel()

	result, err := service.SendLike(requestCtx, alice, bob)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf(
			"SendLike() error = %v, want context deadline exceeded",
			err,
		)
	}

	if result != (matching.SendLikeResult{}) {
		t.Errorf("SendLike() result = %+v, want zero value", result)
	}

	if eventIDCalls.Load() != 0 {
		t.Errorf(
			"event ID generator calls = %d, want 0",
			eventIDCalls.Load(),
		)
	}

	reuseCtx, reuseCancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer reuseCancel()

	if err := pool.Ping(reuseCtx); err != nil {
		t.Fatalf("pool.Ping() after cancellation error = %v", err)
	}

	var (
		likeCount   int
		matchCount  int
		outboxCount int
	)

	if err := lockTx.QueryRow(
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
					WHERE event_type = 'match.created'
					  AND payload ->> 'user_low_id' = $1
					  AND payload ->> 'user_high_id' = $2
				)
		`,
		aliceValue,
		bobValue,
	).Scan(
		&likeCount,
		&matchCount,
		&outboxCount,
	); err != nil {
		t.Fatalf("failed to count final state: %v", err)
	}

	if likeCount != 0 {
		t.Errorf("like count = %d, want 0", likeCount)
	}

	if matchCount != 0 {
		t.Errorf("match count = %d, want 0", matchCount)
	}

	if outboxCount != 0 {
		t.Errorf("outbox count = %d, want 0", outboxCount)
	}
}

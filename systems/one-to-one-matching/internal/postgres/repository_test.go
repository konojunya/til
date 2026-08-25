//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

func TestRepositorySaveLikeIsIdempotent(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(ctx, `INSERT INTO users (id) VALUES ('repository-alice'), ('repository-bob')`); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	alice, err := matching.NewUserID("repository-alice")
	if err != nil {
		t.Fatalf("failed to create user ID for Alice: %v", err)
	}

	bob, err := matching.NewUserID("repository-bob")
	if err != nil {
		t.Fatalf("failed to create user ID for Bob: %v", err)
	}

	like, err := matching.NewLike(alice, bob)
	if err != nil {
		t.Fatalf("failed to create like: %v", err)
	}

	repo := NewRepository(tx)

	created, err := repo.SaveLike(ctx, like)
	if err != nil {
		t.Fatalf("failed to save like: %v", err)
	}

	if !created {
		t.Fatalf("expected like to be created, but it was not")
	}

	created, err = repo.SaveLike(ctx, like)
	if err != nil {
		t.Fatalf("failed to save like again: %v", err)
	}

	if created {
		t.Fatalf("expected like to not be created again, but it was")
	}

	var got int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM likes WHERE sender_id = 'repository-alice' AND receiver_id = 'repository-bob'`).Scan(&got)
	if err != nil {
		t.Fatalf("failed to count likes: %v", err)
	}

	if got != 1 {
		t.Fatalf("expected 1 like, but got %d", got)
	}
}

func TestRepositoryHasLikeKeepsDirection(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(ctx, `INSERT INTO users (id) VALUES ($1), ($2)`, "has-like-alice", "has-like-bob"); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	alice, err := matching.NewUserID("has-like-alice")
	if err != nil {
		t.Fatalf("failed to create user ID for Alice: %v", err)
	}

	bob, err := matching.NewUserID("has-like-bob")
	if err != nil {
		t.Fatalf("failed to create user ID for Bob: %v", err)
	}

	forward, err := matching.NewLike(alice, bob)
	if err != nil {
		t.Fatalf("failed to create forward like: %v", err)
	}

	reverse, err := matching.NewLike(bob, alice)
	if err != nil {
		t.Fatalf("failed to create reverse like: %v", err)
	}

	repo := NewRepository(tx)

	if _, err := repo.SaveLike(ctx, forward); err != nil {
		t.Fatalf("failed to save forward like: %v", err)
	}

	exists, err := repo.HasLike(ctx, forward)
	if err != nil {
		t.Fatalf("failed to check forward like existence: %v", err)
	}
	if !exists {
		t.Fatalf("expected forward like to exist, but it does not")
	}

	exists, err = repo.HasLike(ctx, reverse)
	if err != nil {
		t.Fatalf("failed to check reverse like existence: %v", err)
	}

	if exists {
		t.Fatalf("expected reverse like to not exist, but it does")
	}
}

func TestRepositorySaveMatchIsIdempotent(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO users (id) VALUES ($1), ($2)`,
		"save-match-alice",
		"save-match-bob",
	); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	alice, err := matching.NewUserID("save-match-alice")
	if err != nil {
		t.Fatalf("failed to create Alice: %v", err)
	}

	bob, err := matching.NewUserID("save-match-bob")
	if err != nil {
		t.Fatalf("failed to create Bob: %v", err)
	}

	// 逆順で渡しても、Pairがlow/highへ正規化する。
	pair, err := matching.NewPair(bob, alice)
	if err != nil {
		t.Fatalf("failed to create pair: %v", err)
	}

	repo := NewRepository(tx)

	created, err := repo.SaveMatch(ctx, pair)
	if err != nil {
		t.Fatalf("failed to save match: %v", err)
	}
	if !created {
		t.Fatal("expected match to be created")
	}

	created, err = repo.SaveMatch(ctx, pair)
	if err != nil {
		t.Fatalf("failed to save match again: %v", err)
	}
	if created {
		t.Fatal("expected match to not be created again")
	}

	var got int
	err = tx.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM matches
			WHERE user_low_id = $1
			  AND user_high_id = $2
		`,
		pair.Low().String(),
		pair.High().String(),
	).Scan(&got)
	if err != nil {
		t.Fatalf("failed to count matches: %v", err)
	}

	if got != 1 {
		t.Fatalf("expected 1 match, but got %d", got)
	}
}

func TestRepositoryListMatchesReturnsOnlyUserMatchesInStableOrder(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(context.Background())

	const (
		lowerValue           = "list-matches-a"
		targetValue          = "list-matches-m"
		higherValue          = "list-matches-z"
		unrelatedLowerValue  = "list-matches-x"
		unrelatedHigherValue = "list-matches-y"
	)

	if _, err := tx.Exec(
		ctx,
		`
			INSERT INTO users (id)
			VALUES ($1), ($2), ($3), ($4), ($5)
		`,
		lowerValue,
		targetValue,
		higherValue,
		unrelatedLowerValue,
		unrelatedHigherValue,
	); err != nil {
		t.Fatalf("failed to insert users: %v", err)
	}

	// 期待する取得順とは逆に保存し、結果がINSERT順へ依存しないことを確認する。
	if _, err := tx.Exec(
		ctx,
		`
			INSERT INTO matches (user_low_id, user_high_id)
			VALUES
				($1, $2),
				($3, $4),
				($5, $6)
		`,
		targetValue,
		higherValue,
		unrelatedLowerValue,
		unrelatedHigherValue,
		lowerValue,
		targetValue,
	); err != nil {
		t.Fatalf("failed to insert matches: %v", err)
	}

	target, err := matching.NewUserID(targetValue)
	if err != nil {
		t.Fatalf("failed to create target UserID: %v", err)
	}

	repo := NewRepository(tx)

	got, err := repo.ListMatches(ctx, target)
	if err != nil {
		t.Fatalf("ListMatches() error = %v", err)
	}

	want := [][2]string{
		{lowerValue, targetValue},
		{targetValue, higherValue},
	}

	if len(got) != len(want) {
		t.Fatalf("ListMatches() count = %d, want %d", len(got), len(want))
	}

	for i, pair := range got {
		gotIDs := [2]string{
			pair.Low().String(),
			pair.High().String(),
		}

		if gotIDs != want[i] {
			t.Errorf(
				"ListMatches()[%d] = %q, want %q",
				i,
				gotIDs,
				want[i],
			)
		}
	}
}

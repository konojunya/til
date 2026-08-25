package matching_test

import (
	"context"
	"testing"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

func mustPair(t *testing.T, first, second matching.UserID) matching.Pair {
	t.Helper()

	pair, err := matching.NewPair(first, second)
	if err != nil {
		t.Fatalf("NewPair() error = %v", err)
	}

	return pair
}

func TestMemoryRepositoryStoresLikesByDirection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := matching.NewMemoryRepository()

	alice := mustUserID(t, "alice")
	bob := mustUserID(t, "bob")
	forward := mustLike(t, alice, bob)
	reverse := mustLike(t, bob, alice)

	created, err := repository.SaveLike(ctx, forward)
	if err != nil {
		t.Fatalf("SaveLike() error = %v", err)
	}

	if !created {
		t.Fatal("first SaveLike() created = false, want true")
	}

	created, err = repository.SaveLike(ctx, forward)
	if err != nil {
		t.Fatalf("second SaveLike() error = %v", err)
	}

	if created {
		t.Fatal("second SaveLike() created = true, want false")
	}

	exists, err := repository.HasLike(ctx, forward)
	if err != nil {
		t.Fatalf("HasLike(forward) error = %v", err)
	}

	if !exists {
		t.Fatal("HasLike(forward) = false, want true")
	}

	exists, err = repository.HasLike(ctx, reverse)
	if err != nil {
		t.Fatalf("HasLike(reverse) error = %v", err)
	}

	if exists {
		t.Fatal("HasLike(reverse) = true, want false")
	}
}

func TestMemoryRepositoryStoresMatchOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := matching.NewMemoryRepository()

	alice := mustUserID(t, "alice")
	bob := mustUserID(t, "bob")
	forward := mustPair(t, alice, bob)
	reverse := mustPair(t, bob, alice)

	created, err := repository.SaveMatch(ctx, forward)
	if err != nil {
		t.Fatalf("SaveMatch(forward) error = %v", err)
	}
	if !created {
		t.Fatal("first SaveMatch() created = false, want true")
	}

	created, err = repository.SaveMatch(ctx, reverse)
	if err != nil {
		t.Fatalf("SaveMatch(reverse) error = %v", err)
	}
	if created {
		t.Fatal("second SaveMatch() created = true, want false")
	}

	exists, err := repository.HasMatch(ctx, reverse)
	if err != nil {
		t.Fatalf("HasMatch(reverse) error = %v", err)
	}
	if !exists {
		t.Fatal("HasMatch(reverse) = false, want true")
	}
}

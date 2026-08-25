package matching_test

import (
	"context"
	"testing"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

func TestServiceSendLikeStoresOneWayLikeWithoutMatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := matching.NewMemoryRepository()
	service := matching.NewService(repository)

	alice := mustUserID(t, "alice")
	bob := mustUserID(t, "bob")

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

	like := mustLike(t, alice, bob)
	likeExists, err := repository.HasLike(ctx, like)
	if err != nil {
		t.Fatalf("HasLike() error = %v", err)
	}
	if !likeExists {
		t.Fatal("HasLike() = false, want true")
	}

	pair := mustPair(t, alice, bob)
	matchExists, err := repository.HasMatch(ctx, pair)
	if err != nil {
		t.Fatalf("HasMatch() error = %v", err)
	}
	if matchExists {
		t.Fatal("HasMatch() = true, want false")
	}
}

func TestServiceSendLikeCreatesMatchForMutualLikes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := matching.NewMemoryRepository()
	service := matching.NewService(repository)

	alice := mustUserID(t, "alice")
	bob := mustUserID(t, "bob")

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

	pair := mustPair(t, alice, bob)
	matchExists, err := repository.HasMatch(ctx, pair)
	if err != nil {
		t.Fatalf("HasMatch() error = %v", err)
	}
	if !matchExists {
		t.Fatal("HasMatch() = false, want true")
	}
}

func TestServiceSendLikeIsIdempotentAfterMatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repository := matching.NewMemoryRepository()
	service := matching.NewService(repository)

	alice := mustUserID(t, "alice")
	bob := mustUserID(t, "bob")

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

	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.SendLike(ctx, tt.sender, tt.receiver)
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

	pair := mustPair(t, alice, bob)
	matchExists, err := repository.HasMatch(ctx, pair)
	if err != nil {
		t.Fatalf("HasMatch() error = %v", err)
	}
	if !matchExists {
		t.Fatal("HasMatch() = false, want true")
	}
}

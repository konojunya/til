package matching_test

import (
	"errors"
	"testing"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

func mustLike(t *testing.T, sender, receiver matching.UserID) matching.Like {
	t.Helper()

	like, err := matching.NewLike(sender, receiver)
	if err != nil {
		t.Fatalf("NewLike() error = %v", err)
	}

	return like
}

func TestNewLikePreservesDirection(t *testing.T) {
	t.Parallel()

	alice := mustUserID(t, "alice")
	bob := mustUserID(t, "bob")

	forward, err := matching.NewLike(alice, bob)
	if err != nil {
		t.Fatalf("NewLike(alice, bob) error = %v", err)
	}

	reverse, err := matching.NewLike(bob, alice)
	if err != nil {
		t.Fatalf("NewLike(bob, alice) error = %v", err)
	}

	if got, want := forward.Sender(), alice; got != want {
		t.Fatalf("forward.Sender() = %v, want = %v", got, want)
	}

	if got, want := forward.Receiver(), bob; got != want {
		t.Fatalf("forward.Receiver() = %v, want = %v", got, want)
	}

	if got, want := reverse.Sender(), bob; got != want {
		t.Fatalf("reverse.Sender() = %v, want = %v", got, want)
	}

	if got, want := reverse.Receiver(), alice; got != want {
		t.Fatalf("forward.Receiver() = %v, want = %v", got, want)
	}
}

func TestNewLikeRejectsInvalidLike(t *testing.T) {
	t.Parallel()

	alice := mustUserID(t, "alice")
	var empty matching.UserID

	tests := []struct {
		name     string
		sender   matching.UserID
		receiver matching.UserID
		wantErr  error
	}{
		{
			name:     "sender is empty",
			sender:   empty,
			receiver: alice,
			wantErr:  matching.ErrEmptyUserID,
		},
		{
			name:     "receiver is empty",
			sender:   alice,
			receiver: empty,
			wantErr:  matching.ErrEmptyUserID,
		},
		{
			name:     "sender and receiver are the same user",
			sender:   alice,
			receiver: alice,
			wantErr:  matching.ErrSelfLike,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := matching.NewLike(tt.sender, tt.receiver)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewLike() error = %v, want = %v", err, tt.wantErr)
			}
		})
	}
}

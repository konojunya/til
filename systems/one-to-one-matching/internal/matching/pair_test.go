package matching_test

import (
	"errors"
	"testing"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

func mustUserID(t *testing.T, v string) matching.UserID {
	t.Helper()

	id, err := matching.NewUserID(v)
	if err != nil {
		t.Fatalf("NewUserID(%q) = error = %v", v, err)
	}

	return id
}

func TestNewUserID(t *testing.T) {
	t.Parallel()

	t.Run("空でないIDをつくれる", func(t *testing.T) {
		t.Parallel()

		id, err := matching.NewUserID("alice")
		if err != nil {
			t.Fatalf("NewUserID() error = %v", err)
		}

		if got, want := id.String(), "alice"; got != want {
			t.Fatalf("UserID.String() = %q, want = %q", got, want)
		}
	})

	t.Run("空IDを拒否する", func(t *testing.T) {
		t.Parallel()

		_, err := matching.NewUserID("")
		if !errors.Is(err, matching.ErrEmptyUserID) {
			t.Fatalf("NewUserID() error = %v, want %v", err, matching.ErrEmptyUserID)
		}
	})
}

func TestNewPairNormalizeOrder(t *testing.T) {
	t.Parallel()

	alice := mustUserID(t, "alice")
	bob := mustUserID(t, "bob")

	forward, err := matching.NewPair(alice, bob)
	if err != nil {
		t.Fatalf("NewPair(alice, bob) error = %v", err)
	}

	reverse, err := matching.NewPair(bob, alice)
	if err != nil {
		t.Fatalf("NewPair(bob, alice) error = %v", err)
	}

	if forward != reverse {
		t.Fatalf("input order changed Pair identity; forward = %v, reverse = %v", forward, reverse)
	}

	if forward.Low() != alice || forward.High() != bob {
		t.Fatalf("NewPair() = (%v, %v), want (alice, bob)", forward.Low(), forward.High())
	}
}

func TestNewPairRejectsInvalidPair(t *testing.T) {
	t.Parallel()

	alice := mustUserID(t, "alice")
	var empty matching.UserID

	tests := []struct {
		name    string
		first   matching.UserID
		second  matching.UserID
		wantErr error
	}{
		{
			name:    "first ID is empty",
			first:   empty,
			second:  alice,
			wantErr: matching.ErrEmptyUserID,
		},
		{
			name:    "second ID is empty",
			first:   alice,
			second:  empty,
			wantErr: matching.ErrEmptyUserID,
		},
		{
			name:    "same user",
			first:   alice,
			second:  alice,
			wantErr: matching.ErrSameUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := matching.NewPair(tt.first, tt.second)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewPair() error = %v, want = %v", err, tt.wantErr)
			}
		})
	}

}

func TestPairDistinguishesDifferentUsers(t *testing.T) {
	t.Parallel()

	alice := mustUserID(t, "alice")
	bob := mustUserID(t, "bob")
	carol := mustUserID(t, "carol")

	aliceAndBob, err := matching.NewPair(alice, bob)
	if err != nil {
		t.Fatalf("NewPair(alice, bob) error = %v", err)
	}

	aliceAndCarol, err := matching.NewPair(alice, carol)
	if err != nil {
		t.Fatalf("NewPair(alice, carol) error = %v", err)
	}

	if aliceAndBob == aliceAndCarol {
		t.Fatal("different users produced the same Pair")
	}
}

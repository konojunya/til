package postgres

import (
	"context"
	"fmt"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
	sqldb "github.com/konojunya/til/systems/one-to-one-matching/internal/postgres/db"
)

func (r *Repository) LockMatchingPair(ctx context.Context, first, second matching.UserID) error {
	err := r.queries.LockMatchingPair(ctx, sqldb.LockMatchingPairParams{
		FirstUserID:  first.String(),
		SecondUserID: second.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to lock matching pair: %w", err)
	}

	return nil
}

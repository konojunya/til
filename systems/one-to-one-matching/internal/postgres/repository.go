package postgres

import (
	"context"
	"fmt"

	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
	sqldb "github.com/konojunya/til/systems/one-to-one-matching/internal/postgres/db"
)

type Repository struct {
	queries sqldb.Querier
}

func NewRepository(db sqldb.DBTX) *Repository {
	return newRepository(sqldb.New(db))
}

func newRepository(queries sqldb.Querier) *Repository {
	return &Repository{queries}
}

func (r *Repository) SaveLike(ctx context.Context, like matching.Like) (bool, error) {
	rowsAffected, err := r.queries.SaveLike(ctx, sqldb.SaveLikeParams{
		SenderID:   like.Sender().String(),
		ReceiverID: like.Receiver().String(),
	})
	if err != nil {
		return false, fmt.Errorf("save like: %w", err)
	}

	return rowsAffected == 1, nil
}

func (r *Repository) HasLike(ctx context.Context, like matching.Like) (bool, error) {
	exists, err := r.queries.HasLike(ctx, sqldb.HasLikeParams{
		SenderID:   like.Sender().String(),
		ReceiverID: like.Receiver().String(),
	})
	if err != nil {
		return false, fmt.Errorf("has like: %w", err)
	}

	return exists, nil
}

func (r *Repository) SaveMatch(ctx context.Context, pair matching.Pair) (bool, error) {
	rowsAffected, err := r.queries.SaveMatch(ctx, sqldb.SaveMatchParams{
		UserLowID:  pair.Low().String(),
		UserHighID: pair.High().String(),
	})
	if err != nil {
		return false, fmt.Errorf("save match: %w", err)
	}

	return rowsAffected == 1, nil
}

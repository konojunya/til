package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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
		var postgresError *pgconn.PgError

		// 23503 is the PostgreSQL error code for foreign key violation, which occurs when the referenced user does not exist in the users table.
		if errors.As(err, &postgresError) && postgresError.Code == "23503" {
			return false, fmt.Errorf("save like: %w: %w", matching.ErrUserNotFound, err)
		}

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

func (r *Repository) ListMatches(ctx context.Context, userID matching.UserID) ([]matching.Pair, error) {
	rows, err := r.queries.ListMatches(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list matches: %w", err)
	}

	pairs := make([]matching.Pair, 0, len(rows))

	for i, row := range rows {
		low, err := matching.NewUserID(row.UserLowID)
		if err != nil {
			return nil, fmt.Errorf(
				"list matches row %d low user ID: %w",
				i,
				err,
			)
		}

		high, err := matching.NewUserID(row.UserHighID)
		if err != nil {
			return nil, fmt.Errorf(
				"list matches row %d high user ID: %w",
				i,
				err,
			)
		}

		pair, err := matching.NewPair(low, high)
		if err != nil {
			return nil, fmt.Errorf(
				"list matches row %d pair: %w",
				i,
				err,
			)
		}

		pairs = append(pairs, pair)
	}

	return pairs, nil
}

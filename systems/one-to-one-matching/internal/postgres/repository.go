package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/konojunya/til/systems/one-to-one-matching/internal/matching"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type Repository struct {
	db DBTX
}

func NewRepository(db DBTX) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveLike(ctx context.Context, like matching.Like) (bool, error) {
	commonTag, err := r.db.Exec(ctx, `INSERT INTO likes (sender_id, receiver_id) VALUES ($1, $2) ON CONFLICT (sender_id, receiver_id) DO NOTHING`, like.Sender().String(), like.Receiver())
	if err != nil {
		return false, fmt.Errorf("save like: %w", err)
	}

	return commonTag.RowsAffected() == 1, nil
}

func (r *Repository) HasLike(ctx context.Context, like matching.Like) (bool, error) {
	var exists bool

	// 外側のSELECTにはFROMがないため結果行は常に1行で、EXISTSは内側が0件ならfalse、
	// 1件以上なら行数に関係なくtrueという単一のbooleanへ集約する。
	err := r.db.QueryRow(ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM likes
			WHERE sender_id = $1
			AND receiver_id = $2
		)
		`,
		like.Sender().String(),
		like.Receiver().String(),
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has like: %w", err)
	}

	return exists, nil
}

func (r *Repository) SaveMatch(ctx context.Context, pair matching.Pair) (bool, error) {
	commonTag, err := r.db.Exec(ctx, `INSERT INTO matches (user_low_id, user_high_id) VALUES ($1, $2) ON CONFLICT (user_low_id, user_high_id) DO NOTHING`, pair.Low().String(), pair.High().String())
	if err != nil {
		return false, fmt.Errorf("save match: %w", err)
	}

	return commonTag.RowsAffected() == 1, nil
}

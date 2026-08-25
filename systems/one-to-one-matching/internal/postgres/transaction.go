package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Transactor struct {
	db Beginner
}

func NewTransactor(db Beginner) *Transactor {
	return &Transactor{db: db}
}

func (t *Transactor) WithinTransaction(ctx context.Context, fn func(repo *Repository) error) error {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// callback の失敗や panic, commit 失敗でも transaction を開いたままにしない。
	// commit 後の rollback は pgx.ErrTxClosed になるので、無視してよい
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	repo := NewRepository(tx)

	if err := fn(repo); err != nil {
		return fmt.Errorf("callback failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

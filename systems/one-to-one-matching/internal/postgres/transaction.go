package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	sqldb "github.com/konojunya/til/systems/one-to-one-matching/internal/postgres/db"
)

type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Database interface {
	Beginner
	sqldb.DBTX
}

type Transactor struct {
	database Database
	queries  *sqldb.Queries
}

func NewTransactor(database Database) *Transactor {
	return &Transactor{
		database: database,
		queries:  sqldb.New(database),
	}
}

func (t *Transactor) WithinTransaction(
	ctx context.Context,
	fn func(repo *Repository) error,
) error {
	tx, err := t.database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	repo := newRepository(t.queries.WithTx(tx))

	if err := fn(repo); err != nil {
		return fmt.Errorf("callback failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

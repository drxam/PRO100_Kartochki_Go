package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB обёртка над пулом подключений PostgreSQL.
type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{Pool: pool}
}

func (db *DB) WithTx(ctx context.Context, fn func(tx interface{}) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

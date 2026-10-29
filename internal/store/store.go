package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the subset of pgx used by Store. Both *pgxpool.Pool and pgx.Tx
// satisfy it, so the same queries run inside or outside a transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Store is the typed access layer for the queue database. All SQL lives in
// this package.
type Store struct {
	db   DBTX
	pool *pgxpool.Pool
	inTx bool
}

// New returns a Store that runs queries directly on pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{db: pool, pool: pool}
}

// InTx runs fn inside a transaction. If s is already in a transaction, fn
// joins it instead of opening a nested one.
func (s *Store) InTx(ctx context.Context, fn func(tx *Store) error) error {
	if s.inTx {
		return fn(s)
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&Store{db: tx, pool: s.pool, inTx: true})
	})
}

var (
	// ErrNotFound means the requested row does not exist.
	ErrNotFound = errors.New("not found")

	// ErrLeaseLost means a fenced update matched no row because the caller no
	// longer owns the job.
	ErrLeaseLost = errors.New("lease lost")

	// ErrInvalidState means the operation is not allowed in the job's current
	// state.
	ErrInvalidState = errors.New("invalid state for this operation")
)

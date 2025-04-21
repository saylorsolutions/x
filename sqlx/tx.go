package sqlx

import (
	"context"
	"database/sql"
	"errors"
)

// Beginner is any type that can begin a transaction.
type Beginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// WithTx will use the provided transaction to call the do function.
// If an error is returned from the do function, then the transaction will be rolled back.
// If no error is returned from the do function, then the transaction will be committed.
// The first error returned in the process will be propagated to the caller.
func WithTx(tx *sql.Tx, do func(tx *sql.Tx) error) error {
	if err := do(tx); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

// WithTxCtx will attempt to initiate a transaction with the given Beginner, and pass it to WithTx.
// If the transaction cannot be created, then the error will be returned.
func WithTxCtx(b Beginner, ctx context.Context, opts *sql.TxOptions, do func(tx *sql.Tx) error) error {
	tx, err := b.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	return WithTx(tx, do)
}

// WithTxOpts will do the same thing as WithTxCtx, but will pass [context.Background] as the context.
func WithTxOpts(b Beginner, opts *sql.TxOptions, do func(tx *sql.Tx) error) error {
	return WithTxCtx(b, context.Background(), opts, do)
}

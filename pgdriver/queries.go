package pgdriver

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type conn interface {
	conntx

	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type conntx interface {
	// Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)

	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type queries struct {
}

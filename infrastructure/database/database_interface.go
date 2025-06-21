package database

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// DatabaseInterface defines the methods we need from the database connection
type DatabaseInterface interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	Close()
}

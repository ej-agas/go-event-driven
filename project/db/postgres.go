package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDatabase struct {
	postgres *pgxpool.Pool
}

func NewPostgresDatabase(postgres *pgxpool.Pool) *PostgresDatabase {
	return &PostgresDatabase{postgres: postgres}
}

func (db *PostgresDatabase) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.postgres.Query(ctx, sql, args...)
}

func (db *PostgresDatabase) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.postgres.QueryRow(ctx, sql, args...)
}

func (db *PostgresDatabase) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.postgres.Exec(ctx, sql, args...)
}

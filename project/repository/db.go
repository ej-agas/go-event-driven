package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Database interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

type MockDatabase struct {
	QueryFunc    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRowFunc func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	ExecFunc     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (m *MockDatabase) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return m.QueryFunc(ctx, sql, args...)
}

func (m *MockDatabase) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return m.QueryRowFunc(ctx, sql, args...)
}

func (m *MockDatabase) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return m.ExecFunc(ctx, sql, args...)
}

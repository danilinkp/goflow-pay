package postgres

import (
	"context"
	"fmt"
	"strings"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SelectFactory[T any] struct {
	pool    *pgxpool.Pool
	getter  *trmpgx.CtxGetter
	table   string
	columns []string
}

func NewSelectFactory[T any](pool *pgxpool.Pool, getter *trmpgx.CtxGetter, table string, columns []string) *SelectFactory[T] {
	return &SelectFactory[T]{
		pool:    pool,
		getter:  getter,
		table:   table,
		columns: columns,
	}
}

func (f *SelectFactory[T]) baseQuery() string {
	return fmt.Sprintf("SELECT %s FROM %s", strings.Join(f.columns, ", "), f.table)
}

func (f *SelectFactory[T]) GetOne(ctx context.Context, where string, args ...any) (T, error) {
	var dest T
	conn := f.getter.DefaultTrOrDB(ctx, f.pool)
	query := fmt.Sprintf("%s WHERE %s LIMIT 1", f.baseQuery(), where)

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return dest, err
	}

	return pgx.CollectOneRow(rows, pgx.RowToStructByName[T])
}

func (f *SelectFactory[T]) List(ctx context.Context, where string, args ...any) ([]T, error) {
	query := f.baseQuery()
	if where != "" {
		query = fmt.Sprintf("%s WHERE %s", query, where)
	}

	conn := f.getter.DefaultTrOrDB(ctx, f.pool)
	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}

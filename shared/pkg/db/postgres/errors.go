package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	codeReadOnlySQLTransaction = "25006"
	codeInsufficientPrivilege  = "42501"
)

var ErrReadOnlyInstance = errors.New("read-only instance: writes are not accepted")

func IsReadOnlyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == codeReadOnlySQLTransaction || pgErr.Code == codeInsufficientPrivilege
}

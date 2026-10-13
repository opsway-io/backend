package postgres

import (
	"errors"

	"github.com/jackc/pgconn"
)

var ErrDuplicateEntry = &pgconn.PgError{Code: "23505"}

func IsDuplicateEntryError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == ErrDuplicateEntry.Code
	}
	return false
}

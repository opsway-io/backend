package postgres

import "github.com/jackc/pgconn"

var ErrDuplicateEntry = &pgconn.PgError{Code: "23505"}

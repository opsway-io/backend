package main

import (
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
)

var ErrDuplicateEntry = &pgconn.PgError{Code: "23505"}

func main() {
	// A different Postgres error
	err := &pgconn.PgError{Code: "12345", Message: "Some other error"}

	// Wrapped error
	wrappedErr := fmt.Errorf("wrapped: %w", err)

	if errors.As(wrappedErr, &ErrDuplicateEntry) {
		fmt.Printf("It matched! ErrDuplicateEntry code is now: %s\n", ErrDuplicateEntry.Code)
	} else {
		fmt.Println("It did not match")
	}
}

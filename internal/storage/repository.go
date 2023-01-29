package storage

import (
	"context"
	"errors"
	"io"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
	GetPublicFileURL(bucket string, key string) (url string)
	PutFile(ctx context.Context, bucket string, key string, data io.Reader) (err error)
	DeleteFile(ctx context.Context, bucket string, key string) (err error)
}

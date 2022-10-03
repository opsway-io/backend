package storage

import (
	"context"
	"io"
)

type Repository interface {
	GetPublicFileURL(ctx context.Context, bucket string, key string) (string, error)
	PutFile(ctx context.Context, bucket string, key string, data io.Reader) error
}

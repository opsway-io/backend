package storage

import (
	"context"
	"io"
)

type Service interface {
	GetPublicFileURL(bucket string, key string) (url string)
	PutFile(ctx context.Context, bucket string, key string, data io.Reader) (err error)
	DeleteFile(ctx context.Context, bucket string, key string) (err error)
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetPublicFileURL(bucket string, key string) (url string) {
	return s.repository.GetPublicFileURL(bucket, key)
}

func (s *ServiceImpl) PutFile(ctx context.Context, bucket string, key string, data io.Reader) (err error) {
	return s.repository.PutFile(ctx, bucket, key, data)
}

func (s *ServiceImpl) DeleteFile(ctx context.Context, bucket string, key string) (err error) {
	return s.repository.DeleteFile(ctx, bucket, key)
}

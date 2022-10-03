package storage

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3RepositoryConfig struct {
	Region string
	// TODO
}

type S3Repository struct {
	s3 *s3.Client
}

func NewS3Repository(cfg aws.Config) *S3Repository {
	return &S3Repository{
		s3: s3.NewFromConfig(cfg),
	}
}

func (r *S3Repository) GetPublicFileURL(bucket string, key string) (string, error) {
	return "", nil
}

func (r *S3Repository) PutFile(ctx context.Context, bucket string, key string, data io.Reader) error {
	_, err := r.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})

	return err
}

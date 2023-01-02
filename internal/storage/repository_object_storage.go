package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ObjectStorageRepositoryConfig struct {
	Region      string  `mapstructure:"region"`
	AccessKey   string  `mapstructure:"access_key"`
	SecretKey   string  `mapstructure:"secret_key"`
	EndpointURL *string `mapstructure:"endpoint_url,omitempty"`
	PublicURL   *string `mapstructure:"public_url,omitempty"`
}

type ObjectStorageRepository struct {
	config   ObjectStorageRepositoryConfig
	s3       *s3.Client
	uploader *manager.Uploader
}

func NewObjectStorageRepository(ctx context.Context, conf ObjectStorageRepositoryConfig) *ObjectStorageRepository {
	cfg := aws.Config{
		Region:      conf.Region,
		Credentials: credentials.NewStaticCredentialsProvider(conf.AccessKey, conf.SecretKey, ""),
	}

	if conf.EndpointURL != nil {
		cfg.EndpointResolver = aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               *conf.EndpointURL,
				SigningRegion:     conf.Region,
				HostnameImmutable: true,
			}, nil
		})
	}

	uploader := manager.NewUploader(s3.NewFromConfig(cfg), func(u *manager.Uploader) {
		u.BufferProvider = manager.NewBufferedReadSeekerWriteToPool(25 * 1024 * 1024)
	})

	return &ObjectStorageRepository{
		config:   conf,
		s3:       s3.NewFromConfig(cfg),
		uploader: uploader,
	}
}

func (r *ObjectStorageRepository) GetPublicFileURL(bucket string, key string) string {
	return fmt.Sprintf("%s/%s/%s", *r.config.PublicURL, bucket, key)
}

func (r *ObjectStorageRepository) PutFile(ctx context.Context, bucket string, key string, data io.Reader) error {
	_, err := r.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})

	return err
}

func (r *ObjectStorageRepository) DeleteFile(ctx context.Context, bucket string, key string) error {
	_, err := r.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return err
}

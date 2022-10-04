package storage

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ObjectStorageRepositoryConfig struct {
	Region      string  `mapstructure:"region"`
	AccessKey   string  `mapstructure:"access_key"`
	SecretKey   string  `mapstructure:"secret_key"`
	EndpointURL *string `mapstructure:"endpoint_url,omitempty"`
}

type ObjectStorageRepository struct {
	config ObjectStorageRepositoryConfig
	s3     *s3.Client
}

func NewObjectStorageRepository(ctx context.Context, conf ObjectStorageRepositoryConfig) *ObjectStorageRepository {
	var options []func(*config.LoadOptions) error

	options = append(options, config.WithRegion(conf.Region))
	options = append(options, config.WithCredentialsProvider(
		credentials.NewStaticCredentialsProvider(
			conf.AccessKey,
			conf.SecretKey,
			"",
		),
	))
	if conf.EndpointURL != nil {
		options = append(options, config.WithEndpointResolver(
			aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL: *conf.EndpointURL,
				}, nil
			}),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, options...)
	if err != nil {
		log.Fatal(err)
	}

	return &ObjectStorageRepository{
		config: conf,
		s3:     s3.NewFromConfig(cfg),
	}
}

func (r *ObjectStorageRepository) GetPublicFileURL(bucket string, key string) string {
	return fmt.Sprintf("%s/%s/%s", *r.config.EndpointURL, bucket, key)
}

func (r *ObjectStorageRepository) PutFile(ctx context.Context, bucket string, key string, data io.Reader) error {
	_, err := r.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})

	return err
}

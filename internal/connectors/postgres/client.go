package postgres

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DSN   string `default:"db.sqlite"`
	Debug bool   `default:"false"`
}

func NewClient(ctx context.Context, conf Config) (*gorm.DB, error) {
	dialect := postgres.Open(conf.DSN)

	gormConfig := &gorm.Config{}
	if !conf.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(dialect, gormConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database connection for pinging")
	}

	if err = sqlDB.PingContext(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to ping database")
	}

	return db, nil
}

package clickhouse

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DSN   string `default:"db.clickhouse"`
	Debug bool   `default:"false"`
}

func NewClient(ctx context.Context, conf Config) (*gorm.DB, error) {
	dialect := clickhouse.Open(conf.DSN)

	gormConfig := &gorm.Config{}
	if !conf.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(dialect, gormConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	clickhouseDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database connection for pinging")
	}

	if err = clickhouseDB.PingContext(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to ping database")
	}

	return db, nil
}

package clickhouse

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DSN   string
	Debug bool `default:"false"`
}

func ensureAsyncInsertDSN(dsn string) string {
	if strings.Contains(dsn, "async_insert=") {
		return dsn
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "async_insert=1&wait_for_async_insert=0"
}

func NewClient(ctx context.Context, conf Config) (*gorm.DB, error) {
	gormConfig := &gorm.Config{}
	if conf.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	dsn := ensureAsyncInsertDSN(conf.DSN)

	db, err := gorm.Open(clickhouse.New(clickhouse.Config{
		DSN: dsn,
	}), gormConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	clickhouseDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database connection for pinging")
	}

	clickhouseDB.SetMaxOpenConns(10)
	clickhouseDB.SetMaxIdleConns(5)
	clickhouseDB.SetConnMaxLifetime(time.Hour)

	if err = clickhouseDB.PingContext(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to ping database")
	}

	return db, nil
}

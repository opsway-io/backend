package clickhouse

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string `required:"true"`
	Port     int    `required:"true"`
	User     string `required:"true"`
	Password string `required:"true"`
	Database string `required:"true"`
	Secure   bool   `default:"true"`
	Debug    bool   `default:"false"`
}

func NewClient(ctx context.Context, conf Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("clickhouse+native://%s:%s@%s:%v/%s", conf.User, conf.Password, conf.Host, conf.Port, conf.Database)

	gormConfig := &gorm.Config{}
	if conf.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	} else {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

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

	if err = clickhouseDB.PingContext(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to ping database")
	}

	return db, nil
}

package cmd

import (
	"github.com/mcuadros/go-defaults"
	"github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/connectors/asynq"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/rest"
	"github.com/opsway-io/backend/internal/rest/oauth"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

type Config struct {
	Log            LogConfig                             `mapstructure:"log"`
	Postgres       postgres.Config                       `mapstructure:"postgres"`
	Asynq          asynq.Config                          `mapstructure:"asynq"`
	Clickhouse     clickhouse.Config                     `mapstructure:"clickhouse"`
	Redis          redis.Config                          `mapstructure:"redis"`
	REST           rest.Config                           `mapstructure:"rest"`
	Authentication authentication.Config                 `mapstructure:"authentication"`
	OAuth          *oauth.Config                         `mapstructure:"oauth"`
	ObjectStorage  storage.ObjectStorageRepositoryConfig `mapstructure:"object_storage"`
	Scheduler      SchedulerConfig                       `mapstructure:"scheduler"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

//nolint:gochecknoglobals
var rootCmd = &cobra.Command{}

//nolint:gochecknoinits
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is config.yaml)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to execute root command")
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	if err := viper.ReadInConfig(); err == nil {
		logrus.Info("Using config file: ", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv()
}

func loadConfig() (*Config, error) {
	var config Config
	defaults.SetDefaults(&config)

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func getLogger(config LogConfig) *logrus.Logger {
	logger := logrus.New()
	lvl, err := logrus.ParseLevel(config.Level)
	if err != nil {
		lvl = logrus.InfoLevel
		logger.WithError(err).Warnf("Failed to parse log level, setting log level to '%s'", lvl)
	}
	logger.SetLevel(lvl)
	switch config.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{})
	default:
		logger.Warn("Unknown log format, setting log format to text")
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	return logger
}

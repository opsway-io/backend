package cmd

import (
	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	auth "github.com/opsway-io/backend/internal/authentication"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/probes/http"
	"github.com/opsway-io/backend/internal/rest"
	"github.com/opsway-io/backend/internal/rest/controllers/authentication"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	// Automatically set GOMAXPROCS to match Linux container CPU quota.
	_ "go.uber.org/automaxprocs"
)

var cfgFile string

type Config struct {
	Log            LogConfig                             `mapstructure:"log"`
	Postgres       postgres.Config                       `mapstructure:"postgres"`
	Clickhouse     clickhouse.Config                     `mapstructure:"clickhouse"`
	Redis          redis.Config                          `mapstructure:"redis"`
	REST           rest.Config                           `mapstructure:"rest"`
	Authentication auth.Config                           `mapstructure:"authentication"`
	OAuth          *authentication.OAuthConfig           `mapstructure:"oauth"`
	ObjectStorage  storage.ObjectStorageRepositoryConfig `mapstructure:"object_storage"`
	Prober         ProberConfig                          `mapstructure:"prober"`
	HTTPProbe      http.Config                           `mapstructure:"http_probe"`
	Email          email.Config                          `mapstructure:"email"`
}

var validate = validator.New()

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

	if err := validate.Struct(config); err != nil {
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

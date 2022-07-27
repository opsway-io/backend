package cmd

import (
	"github.com/opsway-io/backend/internal/connectors/influxdb"
	"github.com/opsway-io/backend/internal/connectors/keydb"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/jwt"
	"github.com/opsway-io/backend/internal/rest"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

type Config struct {
	Log      LogConfig       `mapstructure:"log"`
	Postgres postgres.Config `mapstructure:"postgres"`
	KeyDB    keydb.Config    `mapstructure:"keydb"`
	InfluxDB influxdb.Config `mapstructure:"influxdb"`
	REST     rest.Config     `mapstructure:"rest"`
	JWT      jwt.Config      `mapstructure:"jwt"`
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

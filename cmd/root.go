package cmd

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

const (
	slowQueryThreshold = 5 * time.Second
	shutdownPeriod     = 15 * time.Second
)

type Config struct {
	Log LogConfig `mapstructure:"log"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "monitor",
	Short: "A template go application",
	Long: `This is a template go application
with examples for an http server, a grpc server
and a nats consumer. It also includes examples on how to
add health checks, prometheus metrics and migrations`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to execute root command")
	}
}

func init() { //nolint:gochecknoinits
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is config.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logrus.Info("Using config file: ", viper.ConfigFileUsed())
	}
	viper.AutomaticEnv() // read in environment variables that match
}

func loadConfig() (*Config, error) {
	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func GetLogger(config LogConfig) *logrus.Logger {
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

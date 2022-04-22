package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the http server",
	Long: `Start the example http server
Which is a simple rest api for books.`,
	Run: serve,
}

func init() { //nolint:gochecknoinits
	rootCmd.AddCommand(serveCmd)
}

func serve(cmd *cobra.Command, args []string) {
	config, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	l := GetLogger(config.Log)

	l.Info("ok")
}

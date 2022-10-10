package cmd

import (
	"context"

	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var migrationCmd = &cobra.Command{
	Use: "migrater",
	Run: runMigrations,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(migrationCmd)
}

func runMigrations(cmd *cobra.Command, args []string) {
	conf, err := loadConfig()
	if err != nil {
		panic(err)
	}

	l := getLogger(conf.Log)

	l.WithFields(logrus.Fields{}).Info("Migrating clickhouse")

	ctx := context.Background()

	db, err := clickhouse.NewClient(ctx, conf.Clickhouse)
	if err != nil {
		l.WithError(err).Fatal("Failed to create clickhouse")
	}

	db.AutoMigrate(
		entities.HttpResult{},
	)
}

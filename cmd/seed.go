package cmd

import (
	"context"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/seeds"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var seederCmd = &cobra.Command{
	Use: "seed",
	Run: runSeeder,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(seederCmd)
}

func runSeeder(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	conf, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	l := getLogger(conf.Log)

	db, err := postgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("Failed to create Postgres client")
	}

	seeds.Seed001(db)
}

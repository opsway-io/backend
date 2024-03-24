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

	Args: cobra.MinimumNArgs(1),
	ValidArgs: []string{
		"team_opsway",
		"teams_and_users",
		"monitors",
	},
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

	seeders := getSeeders(args)

	for _, s := range seeders {
		s(db)
	}
}

func getSeeders(args []string) []seeds.Seeder {
	var res []seeds.Seeder

	for _, arg := range args {
		switch arg {
		case "team_opsway":
			res = append(res, seeds.TeamOpsway)
		case "teams_and_users":
			res = append(res, seeds.TeamsAndUsers)
		case "monitors":
			res = append(res, seeds.Monitors)
		}
	}

	return res
}

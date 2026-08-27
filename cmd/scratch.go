package cmd

import (
	"context"
	"fmt"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/spf13/cobra"
)

var scratchCmd = &cobra.Command{
	Use: "scratch",
	Run: runScratch,
}

func init() {
	rootCmd.AddCommand(scratchCmd)
}

func runScratch(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Hardcode config for local DB
	cfg := postgres.Config{
		DSN: "host=localhost user=postgres password=pass dbname=opsway port=5432 sslmode=disable",
	}

	db, err := postgres.NewClient(ctx, cfg)
	if err != nil {
		fmt.Printf("failed to connect to db: %v\n", err)
		return
	}

	var incidents []entities.Incident
	if err := db.Order("created_at desc").Limit(10).Find(&incidents).Error; err != nil {
		fmt.Printf("failed to query incidents: %v\n", err)
		return
	}

	for _, inc := range incidents {
		rca := "<nil>"
		if inc.RootCauseAnalysis != nil {
			rca = *inc.RootCauseAnalysis
		}
		fmt.Printf("ID: %d | Title: %s | RCA: %s\n", inc.ID, inc.Title, rca)
	}
}

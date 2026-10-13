package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
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

	dsn := "host=localhost user=postgres password=pass dbname=opsway port=5432 sslmode=disable"
	if envDsn := os.Getenv("PG_DSN"); envDsn != "" {
		dsn = envDsn
	}
	cfg := postgres.Config{
		DSN: dsn,
	}

	db, err := postgres.NewClient(ctx, cfg)
	if err != nil {
		fmt.Printf("failed to connect to db: %v\n", err)
		return
	}

	redisHost := "localhost"
	if envHost := os.Getenv("REDIS_HOST"); envHost != "" {
		redisHost = envHost
	}
	redisPort := uint32(6379)
	if envPort := os.Getenv("REDIS_PORT"); envPort != "" {
		if p, err := strconv.ParseUint(envPort, 10, 32); err == nil {
			redisPort = uint32(p)
		}
	}

	redisCfg := redis.Config{
		Host: redisHost,
		Port: redisPort,
	}
	rdb, err := redis.NewClient(ctx, redisCfg)
	if err != nil {
		fmt.Printf("failed to connect to redis: %v\n", err)
		return
	}

	schedule := monitor.NewSchedule(rdb)

	var monitors []entities.Monitor
	if err := db.Preload("Settings").Preload("Steps").Preload("Steps.Assertions").Preload("Steps.Variables").Where("state = ?", entities.MonitorStateActive).Find(&monitors).Error; err != nil {
		fmt.Printf("failed to query monitors: %v\n", err)
		return
	}

	fmt.Printf("Found %d active monitors\n", len(monitors))
	for _, m := range monitors {
		if len(m.Steps) == 0 {
			fmt.Printf("Monitor %d has no steps in DB!\n", m.ID)
		} else {
			fmt.Printf("Rescheduling monitor %d\n", m.ID)
			mCopy := m
			_ = schedule.Remove(ctx, &mCopy)
			if err := schedule.Add(ctx, &mCopy); err != nil {
				fmt.Printf("failed to add monitor %d: %v\n", m.ID, err)
			}
		}
	}
}

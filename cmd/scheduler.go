package cmd

import (
	"context"

	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type SchedulerConfig struct {
	AvailableLocations []string `mapstructure:"available_locations"`
}

//nolint:gochecknoglobals
var schedulerCmd = &cobra.Command{
	Use: "scheduler",
	Run: runScheduler,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(schedulerCmd)
}

func runScheduler(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	conf, err := loadConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	l := getLogger(conf.Log)

	l.WithFields(logrus.Fields{
		"host": conf.Redis.Host,
		"port": conf.Redis.Port,
		"db":   conf.Redis.DB,
	}).Info("Connecting to redis")

	redisClient, err := connectorRedis.NewClient(ctx, conf.Redis)
	if err != nil {
		l.WithError(err).Fatal("failed to connect to redis")
	}

	schedule := monitor.NewSchedule(redisClient)

	eventService, err := event.NewService(redisClient, "")
	if err != nil {
		l.WithError(err).Fatal("Failed to create event service")
	}

	locations := conf.Prober.AvailableLocations
	if len(locations) == 0 {
		locations = []string{"global", "eu-central-1", "us-east-1"} // Fallback
	}

	l.Info("Starting scheduler for locations: ", locations)

	for _, location := range locations {
		loc := location // capture loop variable
		if err := schedule.On(ctx, loc, func(ctx context.Context, m *entities.Monitor) {
			l.WithFields(logrus.Fields{
				"monitor_id": m.ID,
				"location":   loc,
			}).Debug("Scheduling probe")

			eventService.Publish(events.ProberTask{
				Monitor:  m,
				Location: loc,
			})
		}); err != nil {
			l.WithError(err).Fatalf("failed to start schedule for location %s", loc)
		}
	}

	l.Info("Scheduler running. Waiting for tasks...")
	<-ctx.Done()
	l.Info("Shutting down scheduler...")
}

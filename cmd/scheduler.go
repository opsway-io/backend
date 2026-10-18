package cmd

import (
	"context"
	"sync"
	"time"

	connectorPostgres "github.com/opsway-io/backend/internal/connectors/postgres"
	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/monitor"
	proberPkg "github.com/opsway-io/backend/internal/prober"
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

	// Connect to postgres to sync active monitors on startup
	db, err := connectorPostgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("failed to connect to postgres")
	}

	syncActiveMonitors := func() {
		var monitors []entities.Monitor
		if err := db.WithContext(ctx).
			Preload("Settings").
			Preload("Steps").
			Preload("Steps.Assertions").
			Preload("Steps.Variables").
			Where("state = ?", entities.MonitorStateActive).
			Find(&monitors).Error; err != nil {
			l.WithError(err).Error("failed to query active monitors for schedule")
			return
		}

		l.Infof("Syncing %d active monitors to schedule", len(monitors))
		for _, m := range monitors {
			if len(m.Steps) == 0 {
				continue
			}
			mCopy := m
			if len(mCopy.Settings.Locations) == 0 {
				mCopy.Settings.Locations = []string{monitor.DefaultLocation}
				if mCopy.Settings.ID != 0 {
					_ = db.WithContext(ctx).Model(&mCopy.Settings).Update("locations", mCopy.Settings.Locations).Error
				}
			}
			_ = schedule.Remove(ctx, &mCopy)
			if err := schedule.Add(ctx, &mCopy); err != nil {
				l.WithError(err).Errorf("failed to add monitor %d to schedule", m.ID)
			}
		}
	}

	// Initial startup sync
	syncActiveMonitors()

	// Periodic sync every 5 minutes in background to recover if Redis restarts
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncActiveMonitors()
			}
		}
	}()

	proberService := proberPkg.NewService(redisClient, nil)

	runningLocations := make(map[string]bool)
	var mu sync.Mutex

	startLocationWorker := func(loc string) {
		mu.Lock()
		if runningLocations[loc] {
			mu.Unlock()
			return
		}
		runningLocations[loc] = true
		mu.Unlock()

		l.Infof("Starting scheduler worker for location: %s", loc)
		go func(loc string) {
			for {
				if err := ctx.Err(); err != nil {
					return
				}

				if err := schedule.On(ctx, loc, func(ctx context.Context, m *entities.Monitor) {
					targetLoc := resolveTargetLocation(ctx, proberService, loc, conf.Prober.Location)

					l.WithFields(logrus.Fields{
						"monitor_id": m.ID,
						"location":   targetLoc,
					}).Debug("Scheduling probe")

					eventService.Publish(events.ProberTask{
						Monitor:  m,
						Location: targetLoc,
					})
				}); err != nil {
					l.WithError(err).Errorf("schedule for location %s exited with error, retrying in 2 seconds...", loc)
					select {
					case <-ctx.Done():
						return
					case <-time.After(2 * time.Second):
					}
				}
			}
		}(loc)
	}

	// Initial locations from config or fallbacks
	initialLocations := []string{monitor.DefaultLocation, monitor.LegacyLocation}
	if conf.Prober.Location != "" {
		initialLocations = append(initialLocations, conf.Prober.Location)
	}
	initialLocations = append(initialLocations, conf.Prober.AvailableLocations...)

	for _, loc := range initialLocations {
		startLocationWorker(loc)
	}

	// Periodically discover active probers and start workers dynamically
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				activeLocs, err := proberService.GetActiveLocations(ctx)
				if err != nil {
					continue
				}
				for _, loc := range activeLocs {
					startLocationWorker(loc.ID)
				}
			}
		}
	}()

	l.Info("Scheduler running. Waiting for tasks...")
	<-ctx.Done()
	l.Info("Shutting down scheduler...")
}

func resolveTargetLocation(ctx context.Context, proberService proberPkg.Service, loc string, configuredLoc string) string {
	if proberService != nil {
		if activeLocs, err := proberService.GetActiveLocations(ctx); err == nil && len(activeLocs) > 0 {
			for _, active := range activeLocs {
				if active.ID == loc {
					return loc
				}
			}
			if loc == monitor.LegacyLocation || loc == monitor.DefaultLocation {
				return activeLocs[0].ID
			}
		}
	}

	if loc == monitor.LegacyLocation {
		if configuredLoc != "" && configuredLoc != monitor.LegacyLocation {
			return configuredLoc
		}
		return monitor.DefaultLocation
	}

	return loc
}


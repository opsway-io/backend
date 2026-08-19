package cmd

import (
	"encoding/json"

	"github.com/gammazero/workerpool"
	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/connectors/clickhouse"
	"github.com/opsway-io/backend/internal/connectors/postgres"
	connectorRedis "github.com/opsway-io/backend/internal/connectors/redis"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/report"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"

	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var reportCmd = &cobra.Command{
	Use: "report",
	Run: runReport,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) {
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

	l.WithField("endpoints", conf.Postgres.DSN).Info("Connecting to postgres")

	db, err := postgres.NewClient(ctx, conf.Postgres)
	if err != nil {
		l.WithError(err).Fatal("Failed to connect to postgres")
	}

	ch_db, err := clickhouse.NewClient(ctx, conf.Clickhouse)
	if err != nil {
		l.WithError(err).Fatal("Failed to create clickhouse")
	}

	eventService, err := event.NewService(redisClient)
	if err != nil {
		l.WithError(err).Fatal("Failed to create event service")
	}

	httpResultService := check.NewService(ch_db)
	incidentService := incident.NewService(incident.NewRepository(db), eventService)

	wp := workerpool.New(conf.Report.Concurrency)

	repo := report.NewRepository(db)
	reportService := report.NewService(repo)

	l.WithField("concurrency", conf.Report.Concurrency).Info("Report generator started")

	messages, err := eventService.Subscribe(ctx, "ReportGenerateTask")
	if err != nil {
		l.WithError(err).Fatal("Failed to subscribe to report_tasks")
	}

	for msg := range messages {
		msg := msg

		wp.Submit(func() {
			var task events.ReportGenerateTask
			if err := json.Unmarshal(msg.Payload, &task); err != nil {
				l.WithError(err).Error("failed to unmarshal ReportGenerateTask")
				msg.Ack()
				return
			}

			l.WithField("task", task).Info("Processing report task")

			rep, err := reportService.GetByID(ctx, task.ReportID)
			if err != nil {
				l.WithError(err).Error("failed to fetch report by ID")
				msg.Ack()
				return
			}

			// Do not process already completed/failed tasks (idempotency)
			if rep.Status != entities.ReportStatusPending {
				msg.Ack()
				return
			}

			var reportData entities.ReportData

			// Logic to generate the report data
			switch task.ReportType {
			case "UPTIME", "ALL":
				uptimeReport, err := httpResultService.GetByTeamIDMonitorsUptime(ctx, task.TeamID, task.Start, task.End)
				if err != nil {
					l.WithError(err).Error("failed to get uptime report")
					rep.Status = entities.ReportStatusFailed
					_ = reportService.Update(ctx, rep)
					msg.Ack() // We ack so it doesn't get retried infinitely if it's a persistent error
					return
				}
				reportData.Uptime = uptimeReport
				if task.ReportType != "ALL" {
					break
				}
				fallthrough
			case "PERFORMANCE":
				performanceReport, err := httpResultService.GetByTeamIDMonitorsPerformance(ctx, task.TeamID, task.Start, task.End)
				if err != nil {
					l.WithError(err).Error("failed to get performance report")
					rep.Status = entities.ReportStatusFailed
					_ = reportService.Update(ctx, rep)
					msg.Ack()
					return
				}
				reportData.Performance = performanceReport
				if task.ReportType != "ALL" {
					break
				}
				fallthrough
			case "INCIDENT":
				incidentReport, err := incidentService.GetByTeamIDMonitorsIncidentStats(ctx, task.TeamID, task.Start, task.End)
				if err != nil {
					l.WithError(err).Error("failed to get incident report")
					rep.Status = entities.ReportStatusFailed
					_ = reportService.Update(ctx, rep)
					msg.Ack()
					return
				}
				reportData.Incident = incidentReport
			}

			rep.Report = datatypes.NewJSONType(reportData)
			rep.Status = entities.ReportStatusCompleted

			if err := reportService.Update(ctx, rep); err != nil {
				l.WithError(err).Error("failed to update report status")
				msg.Ack()
				return
			}

			l.WithField("report_id", task.ReportID).Info("Successfully generated report")
			msg.Ack()
		})
	}
	
	l.Info("Shutting down report generator...")
	wp.StopWait()
}

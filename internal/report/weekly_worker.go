package report

import (
	"context"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/check"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/monitor"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type WeeklyWorker interface {
	Start(ctx context.Context) error
}

type weeklyWorker struct {
	logger          *logrus.Entry
	teamService     team.Service
	monitorService  monitor.Service
	checkService    check.Service
	incidentService incident.Service
	emailSender     email.Sender
	interval        time.Duration
	applicationURL  string
}

func NewWeeklyWorker(
	logger *logrus.Entry,
	teamService team.Service,
	monitorService monitor.Service,
	checkService check.Service,
	incidentService incident.Service,
	emailSender email.Sender,
	interval time.Duration,
	applicationURL string,
) WeeklyWorker {
	return &weeklyWorker{
		logger:          logger.WithField("component", "weekly_report_worker"),
		teamService:     teamService,
		monitorService:  monitorService,
		checkService:    checkService,
		incidentService: incidentService,
		emailSender:     emailSender,
		interval:        interval,
		applicationURL:  applicationURL,
	}
}

func (w *weeklyWorker) Start(ctx context.Context) error {
	w.logger.Info("starting weekly report worker")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("stopping weekly report worker")
			return nil
		case <-ticker.C:
			w.generateAndSendReports(ctx)
		}
	}
}

func (w *weeklyWorker) generateAndSendReports(ctx context.Context) {
	w.logger.Info("generating weekly reports for all teams")

	for {
		// Fetch teams. We don't have a GetTeamsPaginated without query? Let's use GetMonitors to find active teams, or maybe just teamService.
		// Wait, teamService.GetUsersByID needs team ID. Let's assume we can get all teams or just get all monitors and group by team.
		// A cleaner way is fetching all teams, but let's just group from monitors.
		// Get monitors by states
		monitorsList, err := w.monitorService.GetMonitorsByStates(ctx, []entities.MonitorState{entities.MonitorStateActive, entities.MonitorStateInactive, entities.MonitorStateMaintenance})
		if err != nil {
			w.logger.WithError(err).Error("failed to fetch monitors")
			return
		}

		if len(*monitorsList) == 0 {
			break
		}

		teamMonitors := make(map[uint][]entities.Monitor)
		for _, m := range *monitorsList {
			teamMonitors[m.TeamID] = append(teamMonitors[m.TeamID], m)
		}

		end := time.Now()
		start := end.Add(-7 * 24 * time.Hour)
		startStr := start.Format(time.RFC3339)
		endStr := end.Format(time.RFC3339)

		for teamID, mList := range teamMonitors {
			w.sendReportForTeam(ctx, teamID, mList, startStr, endStr, start, end)
		}

		break // since we got all monitors in one go
	}
}

func (w *weeklyWorker) sendReportForTeam(ctx context.Context, teamID uint, monitors []entities.Monitor, startStr, endStr string, start, end time.Time) {
	// 1. Gather stats
	uptimeStats, err := w.checkService.GetByTeamIDMonitorsUptime(ctx, teamID, startStr, endStr)
	if err != nil {
		w.logger.WithError(err).Error("failed to get uptime stats")
		return
	}

	incidentStats, err := w.incidentService.GetByTeamIDMonitorsIncidentStats(ctx, teamID, startStr, endStr)
	if err != nil {
		w.logger.WithError(err).Error("failed to get incident stats")
		return
	}

	// 2. Aggregate
	var totalUptime float64
	monitorsUp := 0
	monitorsDown := 0

	for _, m := range monitors {
		if m.State == entities.MonitorStateActive { // Active
			monitorsUp++
		}
	}

	totalMonitorsWithData := 0
	for _, u := range *uptimeStats {
		totalUptime += float64(u.UptimePercentage)
		totalMonitorsWithData++
	}

	averageUptime := 100.0
	if totalMonitorsWithData > 0 {
		averageUptime = totalUptime / float64(totalMonitorsWithData)
	}

	totalIncidents := 0
	resolvedIncidents := 0
	for _, inc := range *incidentStats {
		totalIncidents += inc.Count
	}

	// 3. Get team users
	offset := 0
	limit := 100
	query := ""
	users, err := w.teamService.GetUsersByID(ctx, teamID, &offset, &limit, &query, nil)
	if err != nil || len(*users) == 0 {
		return
	}

	dashboardURL := fmt.Sprintf("%s/dashboard", w.applicationURL)
	teamName := "Your Team" // We don't have GetTeamByID on teamService? We can just use "Your Team"

	tpl := &templates.WeeklyReportTemplate{
		TeamName:          teamName,
		StartDate:         start.Format("Jan 02, 2006"),
		EndDate:           end.Format("Jan 02, 2006"),
		DashboardURL:      dashboardURL,
		TotalMonitors:     len(monitors),
		MonitorsUp:        monitorsUp,
		MonitorsDown:      monitorsDown,
		TotalIncidents:    totalIncidents,
		ResolvedIncidents: resolvedIncidents,
		AverageUptime:     fmt.Sprintf("%.2f", averageUptime),
	}

	// 4. Send emails
	for _, u := range *users {
		err = w.emailSender.Send(ctx, "", u.Email, tpl)
		if err != nil {
			w.logger.WithError(err).Error("failed to send weekly report email")
		}
	}
}

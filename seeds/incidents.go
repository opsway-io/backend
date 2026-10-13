package seeds

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"k8s.io/utils/pointer"
)

type seedIncidentScenario struct {
	source      string
	property    string
	operator    string
	target      string
	description string
	rca         string
}

//nolint:gochecknoglobals
var seedIncidentScenarios = []seedIncidentScenario{
	{"STATUS_CODE", "", "EQUAL", "200", "Expected status code 200 but received 503 Service Unavailable", "Upstream load balancer was draining all targets during a deploy."},
	{"RESPONSE_TIME", "TOTAL", "LESS_THAN", "500", "Response time 2340ms exceeded the 500ms threshold", "Database connection pool exhaustion caused by a slow analytics query."},
	{"JSON_BODY", "$.status", "EQUAL", "ok", "JSON body $.status was \"degraded\", expected \"ok\"", "A dependency health check reported degraded state."},
	{"HEADERS", "content-type", "CONTAINS", "application/json", "Header content-type was text/html, expected application/json", "CDN served a cached maintenance page."},
	{"TLS", "", "NOT_EXPIRED", "", "TLS handshake failed: certificate has expired", "Certificate auto-renewal job failed silently."},
	{"STATUS_CODE", "", "EQUAL", "200", "Target Down: connection refused", "Pod was OOM killed and did not restart in time."},
}

//nolint:gochecknoglobals
var seedIncidentComments = []string{
	"Looking into this now.",
	"Rolled back the latest deploy, monitoring.",
	"Error rate is back to normal.",
	"Root cause identified, follow-up ticket created.",
}

// Incidents seeds a mix of open, acknowledged and resolved incidents over the last 30 days for the
// monitors of the "opsway" team, including occurrences and comments. It is skipped when the team already
// has incidents older than a day (i.e. it was seeded before), so it is safe to run repeatedly.
func Incidents(db *gorm.DB) {
	l := logrus.WithField("seed", "incidents")

	var t entities.Team
	if err := db.Where(entities.Team{Name: "opsway"}).First(&t).Error; err != nil {
		l.WithError(err).Error("team opsway not found, run the team_opsway seed first")

		return
	}

	var existing int64
	db.Model(&entities.Incident{}).
		Where("team_id = ? AND created_at < ?", t.ID, time.Now().Add(-24*time.Hour)).
		Count(&existing)
	if existing > 0 {
		l.WithField("count", existing).Info("team already has historic incidents, skipping")

		return
	}

	var userIDs []uint
	db.Model(&entities.TeamUser{}).
		Where("team_id = ? AND role IN ?", t.ID, []entities.TeamRole{entities.TeamRoleOwner, entities.TeamRoleAdmin}).
		Order("user_id asc").
		Pluck("user_id", &userIDs)
	if len(userIDs) == 0 {
		db.Model(&entities.TeamUser{}).Where("team_id = ?", t.ID).Order("user_id asc").Pluck("user_id", &userIDs)
	}
	if len(userIDs) == 0 {
		l.Error("team has no users, run the team_opsway seed first")

		return
	}

	var monitors []entities.Monitor
	if err := db.Where(entities.Monitor{TeamID: t.ID}).Order("id asc").Limit(8).Find(&monitors).Error; err != nil || len(monitors) == 0 {
		l.WithError(err).Error("team has no monitors, run the monitors seed first")

		return
	}

	rnd := rand.New(rand.NewSource(42)) //nolint:gosec
	now := time.Now()
	created := 0

	for mi, m := range monitors {
		scenario := seedIncidentScenarios[mi%len(seedIncidentScenarios)]

		assertion, err := seedEnsureAssertion(db, m.ID, scenario)
		if err != nil {
			l.WithError(err).WithField("monitor_id", m.ID).Error("failed to ensure assertion")

			continue
		}

		// 2-4 incidents per monitor, oldest first. Only the newest may be unresolved.
		count := 2 + rnd.Intn(3)
		start := now.Add(-30 * 24 * time.Hour)

		for n := 0; n < count; n++ {
			last := n == count-1
			span := now.Sub(start) / time.Duration(count-n)
			createdAt := start.Add(time.Duration(rnd.Int63n(int64(span / 2))))

			if last && mi%3 != 2 {
				// Recent incident for the "open" / "acknowledged" state
				createdAt = now.Add(-time.Duration(1+rnd.Intn(48)) * time.Hour)
			}

			state := "resolved"
			if last {
				switch mi % 3 {
				case 0:
					state = "open"
				case 1:
					state = "acknowledged"
				}
			}

			in := entities.Incident{
				TeamID:              t.ID,
				MonitorID:           pointer.Uint(m.ID),
				MonitorAssertionID:  pointer.Uint(assertion.ID),
				Title:               assertion.Source,
				Description:         pointer.String(scenario.description),
				IsStatusPageVisible: mi%2 == 0,
				CreatedAt:           createdAt,
				UpdatedAt:           createdAt,
			}

			endAt := now
			ackUser := userIDs[rnd.Intn(len(userIDs))]

			if state == "acknowledged" || state == "resolved" {
				ackAt := createdAt.Add(time.Duration(2+rnd.Intn(20)) * time.Minute)
				in.Acknowledged = true
				in.AcknowledgedAt = &ackAt
				in.AcknowledgedBy = &ackUser
				in.UpdatedAt = ackAt
			}

			if state == "resolved" {
				resolvedAt := createdAt.Add(time.Duration(15+rnd.Intn(240)) * time.Minute)
				in.Resolved = true
				in.ResolvedAt = &resolvedAt
				in.ResolvedBy = &ackUser
				in.RootCauseAnalysis = pointer.String(scenario.rca)
				in.UpdatedAt = resolvedAt
				endAt = resolvedAt
			}

			if err := db.Create(&in).Error; err != nil {
				l.WithError(err).WithField("monitor_id", m.ID).Error("failed to create incident")

				continue
			}

			// Occurrences: one per failed check between creation and resolution
			occurrences := 1 + rnd.Intn(15)
			interval := endAt.Sub(createdAt) / time.Duration(occurrences)
			occ := make([]entities.IncidentOccurrence, occurrences)
			for i := range occ {
				occ[i] = entities.IncidentOccurrence{
					IncidentID: in.ID,
					CreatedAt:  createdAt.Add(interval * time.Duration(i)),
				}
			}
			if err := db.Create(&occ).Error; err != nil {
				l.WithError(err).Error("failed to create incident occurrences")
			}

			// Comments on acknowledged/resolved incidents
			if in.Acknowledged {
				comments := 1 + rnd.Intn(len(seedIncidentComments))
				for i := 0; i < comments; i++ {
					at := in.AcknowledgedAt.Add(time.Duration(i*10+1) * time.Minute)
					if err := db.Create(&entities.IncidentComment{
						UserID:     userIDs[rnd.Intn(len(userIDs))],
						IncidentID: in.ID,
						Content:    seedIncidentComments[i],
						CreatedAt:  at,
						UpdatedAt:  at,
					}).Error; err != nil {
						l.WithError(err).Error("failed to create incident comment")
					}
				}
			}

			created++
			start = createdAt.Add(span / 2)
		}
	}

	l.WithField("count", created).Info("seeded incidents")
}

// seedEnsureAssertion returns an assertion of the monitor matching the scenario source, creating one on
// the monitor's first step if missing.
func seedEnsureAssertion(db *gorm.DB, monitorID uint, s seedIncidentScenario) (*entities.MonitorAssertion, error) {
	var a entities.MonitorAssertion
	err := db.Where("monitor_id = ? AND source = ?", monitorID, s.source).Order("id asc").First(&a).Error
	if err == nil {
		return &a, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	a = entities.MonitorAssertion{
		MonitorID: monitorID,
		Source:    s.source,
		Property:  s.property,
		Operator:  s.operator,
		Target:    s.target,
	}

	var step entities.MonitorStep
	if err := db.Where(entities.MonitorStep{MonitorID: monitorID}).Order("order_index asc, id asc").First(&step).Error; err == nil {
		a.MonitorStepID = step.ID
	}

	if err := db.Create(&a).Error; err != nil {
		return nil, fmt.Errorf("failed to create assertion: %w", err)
	}

	return &a, nil
}

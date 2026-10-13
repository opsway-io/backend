package seeds

import (
	"time"

	"github.com/opsway-io/backend/internal/alerting"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

//nolint:gochecknoglobals
var seedAlertRules = []entities.AlertRule{
	{Name: "Page on-call for any incident", Condition: alerting.ConditionAny, Channels: `["email","slack"]`, Enabled: true},
	{Name: "Status code failures", Condition: "STATUS_CODE", Channels: `["email","webhook"]`, Enabled: true},
	{Name: "Slow responses", Condition: "RESPONSE_TIME", Channels: `["slack","microsoft_teams"]`, Enabled: true},
	{Name: "Expiring certificates", Condition: "TLS", Channels: `["email","discord"]`, Enabled: true},
	{Name: "Header mismatches (muted)", Condition: "HEADERS", Channels: `["datadog"]`, Enabled: false},
}

// Alerts seeds alert rules, an escalation policy and notification rules for the "opsway" team, plus
// alert trigger history for the team's incidents (run the incidents seed first). Existing rules are
// matched by name and only get triggers if they have none, so it is safe to run repeatedly.
func Alerts(db *gorm.DB) {
	l := logrus.WithField("seed", "alerts")

	var t entities.Team
	if err := db.Where(entities.Team{Name: "opsway"}).First(&t).Error; err != nil {
		l.WithError(err).Error("team opsway not found, run the team_opsway seed first")

		return
	}

	var incidents []entities.Incident
	if err := db.Where(entities.Incident{TeamID: t.ID}).Order("created_at asc").Find(&incidents).Error; err != nil {
		l.WithError(err).Error("failed to get incidents")

		return
	}

	triggers := 0

	for _, r := range seedAlertRules {
		rule := r
		rule.TeamID = t.ID

		if err := db.Where(entities.AlertRule{TeamID: t.ID, Name: rule.Name}).
			Attrs(entities.AlertRule{Condition: rule.Condition, Channels: rule.Channels, Enabled: rule.Enabled}).
			FirstOrCreate(&rule).Error; err != nil {
			l.WithError(err).WithField("rule", rule.Name).Error("failed to seed alert rule")

			continue
		}

		// Gorm skips zero values on create, so apply the disabled state explicitly
		if !r.Enabled && rule.Enabled {
			db.Model(&rule).Update("enabled", false)
			rule.Enabled = false
		}

		var existing int64
		db.Model(&entities.AlertTrigger{}).Where("alert_rule_id = ?", rule.ID).Count(&existing)
		if existing > 0 || !rule.Enabled {
			continue
		}

		for i := range incidents {
			in := incidents[i]
			if !alerting.Matches(rule.Condition, &in) {
				continue
			}

			incidentID := in.ID
			if err := db.Create(&entities.AlertTrigger{
				TeamID:      t.ID,
				AlertRuleID: rule.ID,
				IncidentID:  &incidentID,
				Channels:    rule.Channels,
				CreatedAt:   in.CreatedAt.Add(5 * time.Second),
			}).Error; err != nil {
				l.WithError(err).Error("failed to seed alert trigger")

				continue
			}

			triggers++
		}
	}

	seedEscalationAndNotificationRules(db, l, t.ID)

	l.WithFields(logrus.Fields{"rules": len(seedAlertRules), "triggers": triggers}).Info("seeded alerts")
}

func seedEscalationAndNotificationRules(db *gorm.DB, l *logrus.Entry, teamID uint) {
	var userIDs []uint
	db.Model(&entities.TeamUser{}).Where("team_id = ?", teamID).Order("user_id asc").Limit(3).Pluck("user_id", &userIDs)
	if len(userIDs) == 0 {
		return
	}

	var count int64
	db.Model(&entities.EscalationPolicy{}).Where("team_id = ?", teamID).Count(&count)
	if count == 0 {
		policy := entities.EscalationPolicy{
			TeamID:                   teamID,
			Name:                     "Primary on-call",
			EscalationTimeoutMinutes: 15,
		}
		for i, id := range userIDs {
			policy.Rotations = append(policy.Rotations, entities.EscalationRotation{UserID: id, Tier: uint(i + 1)})
		}

		if err := db.Create(&policy).Error; err != nil {
			l.WithError(err).Error("failed to seed escalation policy")
		}
	}

	db.Model(&entities.UserNotificationRule{}).Where("user_id = ?", userIDs[0]).Count(&count)
	if count == 0 {
		rules := []entities.UserNotificationRule{
			{UserID: userIDs[0], Channel: "email", Delay: 0},
			{UserID: userIDs[0], Channel: "sms", Delay: 5},
			{UserID: userIDs[0], Channel: "voice", Delay: 15},
		}
		if err := db.Create(&rules).Error; err != nil {
			l.WithError(err).Error("failed to seed notification rules")
		}
	}
}

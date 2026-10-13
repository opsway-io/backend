package alerting

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/sirupsen/logrus"
)

const dispatchTimeout = 30 * time.Second

type Service interface {
	GetRulesByTeamID(ctx context.Context, teamID uint) (*[]entities.AlertRule, error)
	GetRuleByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.AlertRule, error)
	CreateRule(ctx context.Context, rule *entities.AlertRule) error
	UpdateRule(ctx context.Context, rule *entities.AlertRule) error
	DeleteRule(ctx context.Context, teamID, id uint) error

	GetTriggersByRuleID(ctx context.Context, teamID, ruleID uint, offset, limit *int) (*[]entities.AlertTrigger, int64, error)
	GetTriggersByIncidentID(ctx context.Context, teamID, incidentID uint, offset, limit *int) (*[]entities.AlertTrigger, int64, error)

	// OnIncidentCreated evaluates the team's enabled alert rules against a new incident, records a trigger
	// for every matching rule and dispatches notifications asynchronously.
	OnIncidentCreated(ctx context.Context, incident *entities.Incident)
}

type ServiceImpl struct {
	logger     *logrus.Entry
	repository Repository
	dispatcher Dispatcher
}

// NewService creates the alerting service. dispatcher may be nil, in which case triggers are recorded
// but no notifications are sent.
func NewService(logger *logrus.Entry, repository Repository, dispatcher Dispatcher) Service {
	return &ServiceImpl{
		logger:     logger.WithField("module", "alerting"),
		repository: repository,
		dispatcher: dispatcher,
	}
}

func (s *ServiceImpl) GetRulesByTeamID(ctx context.Context, teamID uint) (*[]entities.AlertRule, error) {
	return s.repository.GetRulesByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetRuleByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.AlertRule, error) {
	return s.repository.GetRuleByTeamIDAndID(ctx, teamID, id)
}

func (s *ServiceImpl) CreateRule(ctx context.Context, rule *entities.AlertRule) error {
	rule.Channels = EncodeChannels(ParseChannels(rule.Channels))

	return s.repository.CreateRule(ctx, rule)
}

func (s *ServiceImpl) UpdateRule(ctx context.Context, rule *entities.AlertRule) error {
	rule.Channels = EncodeChannels(ParseChannels(rule.Channels))

	return s.repository.UpdateRule(ctx, rule)
}

func (s *ServiceImpl) DeleteRule(ctx context.Context, teamID, id uint) error {
	return s.repository.DeleteRule(ctx, teamID, id)
}

func (s *ServiceImpl) GetTriggersByRuleID(ctx context.Context, teamID, ruleID uint, offset, limit *int) (*[]entities.AlertTrigger, int64, error) {
	return s.repository.GetTriggersByRuleID(ctx, teamID, ruleID, offset, limit)
}

func (s *ServiceImpl) GetTriggersByIncidentID(ctx context.Context, teamID, incidentID uint, offset, limit *int) (*[]entities.AlertTrigger, int64, error) {
	return s.repository.GetTriggersByIncidentID(ctx, teamID, incidentID, offset, limit)
}

func (s *ServiceImpl) OnIncidentCreated(ctx context.Context, incident *entities.Incident) {
	l := s.logger.WithFields(logrus.Fields{
		"team_id":     incident.TeamID,
		"incident_id": incident.ID,
	})

	rules, err := s.repository.GetEnabledRulesByTeamID(ctx, incident.TeamID)
	if err != nil {
		l.WithError(err).Error("failed to get alert rules")

		return
	}

	for i := range *rules {
		rule := (*rules)[i]

		if !Matches(rule.Condition, incident) {
			continue
		}

		channels := ParseChannels(rule.Channels)
		incidentID := incident.ID

		if err := s.repository.CreateTrigger(ctx, &entities.AlertTrigger{
			TeamID:      incident.TeamID,
			AlertRuleID: rule.ID,
			IncidentID:  &incidentID,
			Channels:    EncodeChannels(channels),
		}); err != nil {
			l.WithError(err).WithField("alert_rule_id", rule.ID).Error("failed to record alert trigger")

			continue
		}

		l.WithField("alert_rule_id", rule.ID).Info("alert rule triggered")

		if s.dispatcher == nil || len(channels) == 0 {
			continue
		}

		in := *incident

		go func() {
			dctx, cancel := context.WithTimeout(context.Background(), dispatchTimeout)
			defer cancel()

			s.dispatcher.Dispatch(dctx, &rule, &in, channels)
		}()
	}
}

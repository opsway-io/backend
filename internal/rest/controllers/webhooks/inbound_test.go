package webhooks_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/rest/controllers/webhooks"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockTeamService struct{}

func (m *mockTeamService) CreateWithOwnerUserID(ctx context.Context, t *entities.Team, ownerUserID uint) error {
	return nil
}
func (m *mockTeamService) GetByID(ctx context.Context, id uint) (*entities.Team, error) {
	return &entities.Team{ID: id, Name: "Test Team"}, nil
}
func (m *mockTeamService) GetByStripeID(ctx context.Context, stripeID string) (*entities.Team, error) {
	return nil, nil
}
func (m *mockTeamService) GetTeamsAndRoleByUserID(ctx context.Context, userID uint) (*[]team.TeamAndRole, error) {
	return nil, nil
}
func (m *mockTeamService) GetUsersByID(ctx context.Context, teamId uint, offset *int, limit *int, query *string, role *entities.TeamRole) (*[]team.TeamUser, error) {
	return nil, nil
}
func (m *mockTeamService) GetTeamUserCount(ctx context.Context, teamId uint) (int64, error) {
	return 0, nil
}
func (m *mockTeamService) GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error) {
	return nil, nil
}
func (m *mockTeamService) UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.TeamRole) error {
	return nil
}
func (m *mockTeamService) UpdateBilling(ctx context.Context, teamID uint, customerID string, plan string) error {
	return nil
}
func (m *mockTeamService) UpdateTeam(ctx context.Context, t *entities.Team) error {
	return nil
}
func (m *mockTeamService) RemoveUser(ctx context.Context, teamID, userID uint) error {
	return nil
}
func (m *mockTeamService) UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error {
	return nil
}
func (m *mockTeamService) Delete(ctx context.Context, id uint) error {
	return nil
}
func (m *mockTeamService) UploadAvatar(ctx context.Context, teamID uint, file io.Reader) error {
	return nil
}
func (m *mockTeamService) DeleteAvatar(ctx context.Context, teamID uint) error {
	return nil
}
func (m *mockTeamService) GetAvatarURLByID(teamID uint) (URL string) {
	return ""
}
func (m *mockTeamService) IsNameAvailable(ctx context.Context, name string) (bool, error) {
	return true, nil
}
func (m *mockTeamService) GetTeamInvitations(ctx context.Context, teamID uint) (*[]entities.TeamInvitation, error) {
	return nil, nil
}
func (m *mockTeamService) DeleteTeamInvitation(ctx context.Context, teamID uint, email string) error {
	return nil
}
func (m *mockTeamService) InviteByEmail(ctx context.Context, teamID uint, role entities.TeamRole, email string) error {
	return nil
}
func (m *mockTeamService) GenerateInviteLink(ctx context.Context, teamID uint, role entities.TeamRole, email string) (string, error) {
	return "", nil
}
func (m *mockTeamService) AcceptInviteByToken(ctx context.Context, token string, user *entities.User) error {
	return nil
}

type mockIncidentService struct {
	createdIncidents []entities.Incident
}

func (m *mockIncidentService) GetByID(ctx context.Context, id uint) (*entities.Incident, error) {
	return nil, nil
}
func (m *mockIncidentService) GetByTeamIDPaginated(ctx context.Context, teamID uint, offset, limit *int) (*[]entities.Incident, error) {
	return nil, nil
}
func (m *mockIncidentService) GetByTeamIDAndStatusPaginated(ctx context.Context, teamID uint, resolved *bool, offset, limit *int) (*[]entities.Incident, error) {
	return nil, nil
}
func (m *mockIncidentService) GetByMonitorIDWithAssertionPaginated(ctx context.Context, monitorID uint, offset, limit *int) (*[]incident.IncidentAndAssertion, error) {
	return nil, nil
}
func (m *mockIncidentService) GetActiveByMonitorIDs(ctx context.Context, monitorIDs []uint) ([]entities.Incident, error) {
	return nil, nil
}
func (m *mockIncidentService) Upsert(ctx context.Context, incidents *[]entities.Incident) error {
	return nil
}
func (m *mockIncidentService) Create(ctx context.Context, incidents *[]entities.Incident) error {
	if incidents != nil {
		m.createdIncidents = append(m.createdIncidents, *incidents...)
	}
	return nil
}
func (m *mockIncidentService) Update(ctx context.Context, incident *entities.Incident) error {
	return nil
}
func (m *mockIncidentService) Delete(ctx context.Context, incident *entities.Incident) error {
	return nil
}
func (m *mockIncidentService) GetByTeamIDMonitorsIncidentStats(ctx context.Context, teamID uint, start, end string) (*[]entities.MonitorIncident, error) {
	return nil, nil
}
func (m *mockIncidentService) CreateOccurrence(ctx context.Context, occurrence *entities.IncidentOccurrence) error {
	return nil
}
func (m *mockIncidentService) GetOccurrencesPaginated(ctx context.Context, incidentID uint, offset, limit *int) (int, *[]entities.IncidentOccurrence, error) {
	return 0, nil, nil
}

func TestInboundWebhooks(t *testing.T) {
	e := echo.New()
	teamSvc := &mockTeamService{}
	incSvc := &mockIncidentService{}
	logger := logrus.NewEntry(logrus.StandardLogger())

	webhooks.Register(e.Group("/v1"), logger, nil, teamSvc, incSvc, nil)

	t.Run("PrometheusAlertmanager", func(t *testing.T) {
		payload := webhooks.PrometheusAlertmanagerPayload{
			Status: "firing",
			Alerts: []webhooks.PrometheusAlert{
				{
					Status: "firing",
					Labels: map[string]string{
						"alertname": "HighMemoryUsage",
					},
					Annotations: map[string]string{
						"description": "Memory is above 90%",
					},
				},
			},
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/prometheus/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
		assert.NotEmpty(t, incSvc.createdIncidents)
		assert.Equal(t, "Prometheus: HighMemoryUsage", incSvc.createdIncidents[len(incSvc.createdIncidents)-1].Title)
	})

	t.Run("CloudWatchAlarm", func(t *testing.T) {
		payload := webhooks.CloudWatchAlarmPayload{
			AlarmName:        "HighCPUUtilization",
			NewStateValue:    "ALARM",
			NewStateReason:   "Threshold Crossed",
			AlarmDescription: "CPU > 80%",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/cloudwatch/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
		assert.Equal(t, "AWS CloudWatch: HighCPUUtilization", incSvc.createdIncidents[len(incSvc.createdIncidents)-1].Title)
	})

	t.Run("GenericInboundAlert", func(t *testing.T) {
		payload := webhooks.GenericInboundAlertPayload{
			Title:       "Payment Gateway Slow",
			Description: "API latency is > 2000ms",
			Status:      "down",
			Source:      "Pingdom",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inbound/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
		assert.Equal(t, "[Pingdom] Payment Gateway Slow", incSvc.createdIncidents[len(incSvc.createdIncidents)-1].Title)
	})
}

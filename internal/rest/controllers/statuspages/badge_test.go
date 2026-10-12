package statuspages_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/rest/controllers/statuspages"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockStatusPageService struct {
	sp *entities.StatusPage
}

func (m *mockStatusPageService) GetByTeamID(ctx context.Context, teamID uint) ([]*entities.StatusPage, error) {
	return nil, nil
}
func (m *mockStatusPageService) GetByIDAndTeamID(ctx context.Context, id, teamID uint) (*entities.StatusPage, error) {
	return nil, nil
}
func (m *mockStatusPageService) GetByDomain(ctx context.Context, domain string) (*entities.StatusPage, error) {
	return m.sp, nil
}
func (m *mockStatusPageService) CountByTeamID(ctx context.Context, teamID uint) (int64, error) {
	return 0, nil
}
func (m *mockStatusPageService) Create(ctx context.Context, statusPage *entities.StatusPage) error {
	return nil
}
func (m *mockStatusPageService) Update(ctx context.Context, statusPage *entities.StatusPage) error {
	return nil
}
func (m *mockStatusPageService) Delete(ctx context.Context, id, teamID uint) error {
	return nil
}
func (m *mockStatusPageService) ReplaceMonitors(ctx context.Context, statusPage *entities.StatusPage, monitors []entities.Monitor) error {
	return nil
}
func (m *mockStatusPageService) ReplaceGroups(ctx context.Context, statusPage *entities.StatusPage, groups []entities.StatusPageGroup) error {
	return nil
}
func (m *mockStatusPageService) Subscribe(ctx context.Context, statusPageID uint, email string, token string) error {
	return nil
}
func (m *mockStatusPageService) SubscribeChannel(ctx context.Context, statusPageID uint, email string, channel string, destination string, token string) error {
	return nil
}
func (m *mockStatusPageService) GetSubscriberByToken(ctx context.Context, token string) (*entities.StatusPageSubscriber, error) {
	return nil, nil
}
func (m *mockStatusPageService) VerifySubscriber(ctx context.Context, token string) error {
	return nil
}
func (m *mockStatusPageService) Unsubscribe(ctx context.Context, token string) error {
	return nil
}
func (m *mockStatusPageService) GetVerifiedSubscribers(ctx context.Context, statusPageID uint) ([]entities.StatusPageSubscriber, error) {
	return nil, nil
}

func TestBadgeAndWidget(t *testing.T) {
	e := echo.New()

	sp := &entities.StatusPage{
		ID:     1,
		Name:   "Acme Status",
		Domain: "acme",
		Monitors: []entities.Monitor{
			{
				ID:    10,
				Name:  "API",
				State: entities.MonitorStateActive,
			},
		},
	}

	svc := &mockStatusPageService{sp: sp}
	logger := logrus.NewEntry(logrus.StandardLogger())
	statuspages.RegisterPublic(e.Group("/v1"), logger, "https://status.opsway.dev", svc, nil, nil, nil, nil)

	t.Run("GetBadgeSVG", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/public/status-pages/acme/badge.svg", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "image/svg+xml")
		assert.Contains(t, rec.Body.String(), "<svg")
		assert.Contains(t, rec.Body.String(), "operational")
	})

	t.Run("GetWidgetSummary", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/public/status-pages/acme/widget.json", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
		assert.Contains(t, rec.Body.String(), `"status":"OPERATIONAL"`)
		assert.Contains(t, rec.Body.String(), `"statusColor":"#4caf50"`)
	})

	t.Run("GetWidgetScript", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/public/status-pages/acme/widget.js", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/javascript")
		assert.Contains(t, rec.Body.String(), "opsway-status-banner")
	})
}

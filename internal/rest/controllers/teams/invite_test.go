package teams_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/rest/controllers/teams"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	teamMocks "github.com/opsway-io/backend/internal/team/mocks"
	userMocks "github.com/opsway-io/backend/internal/user/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUserService struct {
	*userMocks.Service
}

func (m *mockUserService) GetNotificationRules(ctx context.Context, userID uint) ([]entities.UserNotificationRule, error) {
	return nil, nil
}

func (m *mockUserService) SetNotificationRules(ctx context.Context, userID uint, rules []entities.UserNotificationRule) error {
	return nil
}

func TestHandlers_PostTeamInvitesAccept(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	setupTest := func() (*echo.Echo, *teamMocks.Service, *mockUserService, *teams.Handlers) {
		e := echo.New()
		e.Validator = helpers.NewValidator()
		teamSvc := new(teamMocks.Service)
		userSvc := &mockUserService{Service: new(userMocks.Service)}
		h := &teams.Handlers{
			TeamService: teamSvc,
			UserService: userSvc,
		}
		return e, teamSvc, userSvc, h
	}

	makeContext := func(e *echo.Echo, reqBody map[string]interface{}, userID uint) (handlers.AuthenticatedContext, *httptest.ResponseRecorder) {
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/teams/invites/accept", bytes.NewReader(bodyBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		return handlers.AuthenticatedContext{
			Context: c,
			Log:     logger,
			UserID:  userID,
		}, rec
	}

	t.Run("returns 204 No Content when acceptance succeeds", func(t *testing.T) {
		e, teamSvc, userSvc, h := setupTest()
		c, rec := makeContext(e, map[string]interface{}{"token": "valid-token"}, 1)

		user := &entities.User{ID: 1, Email: "user@example.com"}
		userSvc.On("GetUserByID", mock.Anything, uint(1)).Return(user, nil)
		teamSvc.On("AcceptInviteByToken", mock.Anything, "valid-token", user).Return(nil)

		err := h.PostTeamInvitesAccept(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("returns 403 Forbidden when email does not match token", func(t *testing.T) {
		e, teamSvc, userSvc, h := setupTest()
		c, _ := makeContext(e, map[string]interface{}{"token": "mismatched-token"}, 1)

		user := &entities.User{ID: 1, Email: "wrong@example.com"}
		userSvc.On("GetUserByID", mock.Anything, uint(1)).Return(user, nil)
		teamSvc.On("AcceptInviteByToken", mock.Anything, "mismatched-token", user).Return(team.ErrEmailMismatch)

		err := h.PostTeamInvitesAccept(c)
		assert.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
		assert.Equal(t, "this invitation was sent to a different email address", httpErr.Message)
	})

	t.Run("returns 404 Not Found when invite is not found or expired", func(t *testing.T) {
		e, teamSvc, userSvc, h := setupTest()
		c, _ := makeContext(e, map[string]interface{}{"token": "missing-token"}, 1)

		user := &entities.User{ID: 1, Email: "user@example.com"}
		userSvc.On("GetUserByID", mock.Anything, uint(1)).Return(user, nil)
		teamSvc.On("AcceptInviteByToken", mock.Anything, "missing-token", user).Return(team.ErrInvitationNotFound)

		err := h.PostTeamInvitesAccept(c)
		assert.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, httpErr.Code)
		assert.Equal(t, "invitation not found or expired", httpErr.Message)
	})

	t.Run("returns 400 Bad Request when token is invalid", func(t *testing.T) {
		e, teamSvc, userSvc, h := setupTest()
		c, _ := makeContext(e, map[string]interface{}{"token": "malformed-token"}, 1)

		user := &entities.User{ID: 1, Email: "user@example.com"}
		userSvc.On("GetUserByID", mock.Anything, uint(1)).Return(user, nil)
		teamSvc.On("AcceptInviteByToken", mock.Anything, "malformed-token", user).Return(team.ErrInvalidInvitationToken)

		err := h.PostTeamInvitesAccept(c)
		assert.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, httpErr.Code)
	})
}

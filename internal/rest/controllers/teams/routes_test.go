package teams_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/controllers/suite"
	"github.com/opsway-io/backend/internal/rest/controllers/teams"
	teamMocks "github.com/opsway-io/backend/internal/team/mocks"
	userMocks "github.com/opsway-io/backend/internal/user/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestTeamsAndSuiteRouteRegistration(t *testing.T) {
	e := echo.New()
	authRoot := e.Group("/v1")
	logger := logrus.NewEntry(logrus.New())

	teamSvc := new(teamMocks.Service)
	userSvc := &mockUserService{Service: new(userMocks.Service)}

	teamsGroup := teams.Register(authRoot, logger, teamSvc, userSvc, nil, nil)
	suite.Register(authRoot, teamsGroup, logger, nil, nil, nil)

	routes := e.Routes()

	var getTeamRouteFound bool
	var getEntitlementsRouteFound bool
	for _, r := range routes {
		if r.Method == http.MethodGet && r.Path == "/v1/teams/:teamId" {
			// Ensure it is not NotFoundHandler
			assert.NotContains(t, r.Name, "NotFoundHandler")
			getTeamRouteFound = true
		}
		if r.Method == http.MethodGet && r.Path == "/v1/teams/:teamId/entitlements" {
			getEntitlementsRouteFound = true
		}
	}

	assert.True(t, getTeamRouteFound, "GET /v1/teams/:teamId route should be registered and not overwritten by NotFoundHandler")
	assert.True(t, getEntitlementsRouteFound, "GET /v1/teams/:teamId/entitlements route should be registered")
}

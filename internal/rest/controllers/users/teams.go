package users

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
)

type GetUserTeamsRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

type GetUserTeamsResponse struct {
	Teams []GetUserTeamsRequestTeam `json:"teams"`
}

type GetUserTeamsRequestTeam struct {
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	DisplayName *string           `json:"displayName"`
	AvatarURL   *string           `json:"avatarUrl"`
	Role        entities.TeamRole `json:"role"`
}

func (h *Handlers) GetUserTeams(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetUserTeamsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetUserTeamsRequest")

		return echo.ErrBadRequest
	}

	teams, err := h.TeamService.GetTeamsAndRoleByUserID(c.Request().Context(), req.UserID)
	if err != nil {
		if errors.Is(err, team.ErrNotFound) {
			c.Log.WithError(err).Debug("teams not found")

			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to get teams")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, newGetUserTeamsResponse(teams, h.TeamService))
}

func newGetUserTeamsResponse(teams *[]team.TeamAndRole, teamService team.Service) GetUserTeamsResponse {
	var response GetUserTeamsResponse

	for _, team := range *teams {
		t := GetUserTeamsRequestTeam{
			ID:          team.ID,
			Name:        team.Name,
			DisplayName: team.DisplayName,
			Role:        team.Role,
		}

		if team.HasAvatar {
			avatarURL := teamService.GetAvatarURLByID(team.ID)
			t.AvatarURL = &avatarURL
		}

		response.Teams = append(response.Teams, t)
	}

	return response
}

package teams

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	"k8s.io/utils/pointer"
)

type GetTeamRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetTeamResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	DisplayName *string   `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (h *Handlers) GetTeam(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetTeamRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind GetTeamRequest")

		return echo.ErrBadRequest
	}

	team, err := h.TeamService.GetByID(ctx.Request().Context(), req.TeamID)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to get team")

		return echo.ErrInternalServerError
	}

	return ctx.JSON(http.StatusOK, newGetTeamResponse(team, h.TeamService))
}

func newGetTeamResponse(t *entities.Team, teamService team.Service) GetTeamResponse {
	team := GetTeamResponse{
		ID:          t.ID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}

	if t.HasAvatar {
		team.AvatarURL = pointer.StringPtr(teamService.GetAvatarURLByID(t.ID))
	}

	return team
}

type PutTeamRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gt=0"`
	DisplayName string `json:"displayName" validate:"max=255"`
}

func (h *Handlers) PutTeam(ctx hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutTeamRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PutTeamRequest")

		return echo.ErrBadRequest
	}

	if err = h.TeamService.UpdateDisplayName(
		ctx.Request().Context(),
		req.TeamID,
		req.DisplayName,
	); err != nil {
		ctx.Log.WithError(err).Debug("failed to update team")

		return echo.ErrInternalServerError
	}

	return ctx.NoContent(http.StatusNoContent)
}

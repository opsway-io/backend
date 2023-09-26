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
	PaymentPlan string    `json:"paymentPlan"`
	AvatarURL   *string   `json:"avatarUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (h *Handlers) GetTeam(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetTeamRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetTeamRequest")

		return echo.ErrBadRequest
	}

	team, err := h.TeamService.GetByID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get team")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, newGetTeamResponse(team, h.TeamService))
}

func newGetTeamResponse(t *entities.Team, teamService team.Service) GetTeamResponse {
	team := GetTeamResponse{
		ID:          t.ID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		PaymentPlan: t.PaymentPlan,
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

func (h *Handlers) PutTeam(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutTeamRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutTeamRequest")

		return echo.ErrBadRequest
	}

	if err = h.TeamService.UpdateDisplayName(
		c.Request().Context(),
		req.TeamID,
		req.DisplayName,
	); err != nil {
		c.Log.WithError(err).Debug("failed to update team")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type DeleteTeamRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteTeam(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteTeamRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteTeamRequest")

		return echo.ErrBadRequest
	}

	if err = h.TeamService.Delete(c.Request().Context(), req.TeamID); err != nil {
		c.Log.WithError(err).Debug("failed to delete team")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PostTeamRequest struct {
	Name        string  `json:"name" validate:"required,max=255"`
	DisplayName *string `json:"displayName" validate:"max=255"`
}

type PostTeamResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	DisplayName *string   `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (h *Handlers) PostTeam(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostTeamRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostTeamRequest")

		return echo.ErrBadRequest
	}

	t := entities.Team{
		Name:        req.Name,
		DisplayName: req.DisplayName,
	}

	if err := h.TeamService.CreateWithOwnerUserID(c.Request().Context(), &t, c.UserID); err != nil {
		c.Log.WithError(err).Debug("failed to create team")

		return echo.ErrInternalServerError
	}

	res := PostTeamResponse{
		ID:          t.ID,
		Name:        t.Name,
		DisplayName: t.DisplayName,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}

	if t.HasAvatar {
		res.AvatarURL = pointer.StringPtr(h.TeamService.GetAvatarURLByID(t.ID))
	}

	return c.JSON(http.StatusCreated, res)
}

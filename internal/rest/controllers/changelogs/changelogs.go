package changelogs

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetChangelogsRequest struct {
	TeamID uint    `param:"teamId" validate:"required,numeric,gte=0"`
	Offset *int    `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit  *int    `query:"limit" validate:"numeric,gt=0" default:"10"`
	Query  *string `query:"query" validate:"max=255"`
}

type GetChangelogsResponse struct {
	Changelogs []GetChangelogsResponseChangelog `json:"changelogs"`
	TotalCount int                              `json:"totalCount"`
}

type GetChangelogsResponseChangelog struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelogs(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[GetChangelogsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetChangelogsRequest")

		return echo.ErrBadRequest
	}

	changelogs, totalCount, err := h.ChangelogsService.GetAll(ctx, req.TeamID, req.Offset, req.Limit, req.Query)
	if err != nil {
		c.Log.WithError(err).Error("failed to get changelogs")

		return echo.ErrInternalServerError
	}

	resp := h.newGetChangelogsResponse(changelogs, totalCount)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetChangelogsResponse(changelogs []entities.Changelog, totalCount int) GetChangelogsResponse {
	response := make([]GetChangelogsResponseChangelog, len(changelogs))

	for i, changelog := range changelogs {
		response[i] = GetChangelogsResponseChangelog{
			ID:        changelog.ID,
			Name:      changelog.Name,
			CreatedAt: changelog.CreatedAt,
			UpdatedAt: changelog.UpdatedAt,
		}
	}

	return GetChangelogsResponse{
		Changelogs: response,
		TotalCount: totalCount,
	}
}

type PostChangelogsRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gte=0"`
	Name   string `json:"name" validate:"required,max=255,min=1"`
}

func (h *Handlers) PostChangelogs(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[PostChangelogsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostChangelogsRequest")

		return echo.ErrBadRequest
	}

	if _, err := h.ChangelogsService.Create(ctx, req.TeamID, req.Name); err != nil {
		c.Log.WithError(err).Error("failed to create changelog")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}

type GetChangelogRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
}

type GetChangelogResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelog(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[GetChangelogRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetChangelogRequest")

		return echo.ErrBadRequest
	}

	changelog, err := h.ChangelogsService.Get(ctx, req.TeamID, req.ChangelogID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get changelog")

		return echo.ErrInternalServerError
	}

	resp := h.newGetChangelogResponse(changelog)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetChangelogResponse(changelog entities.Changelog) GetChangelogResponse {
	return GetChangelogResponse{
		ID:        changelog.ID,
		Name:      changelog.Name,
		CreatedAt: changelog.CreatedAt,
		UpdatedAt: changelog.UpdatedAt,
	}
}

type DeleteChangelogRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteChangelog(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[DeleteChangelogRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteChangelogRequest")

		return echo.ErrBadRequest
	}

	if err := h.ChangelogsService.Delete(ctx, req.TeamID, req.ChangelogID); err != nil {
		c.Log.WithError(err).Error("failed to delete changelog")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PutChangelogRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint   `param:"changelogId" validate:"required,numeric,gte=0"`
	Name        string `json:"name" validate:"required,max=255"`
}

func (h *Handlers) PutChangelog(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[PutChangelogRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutChangelogRequest")

		return echo.ErrBadRequest
	}

	if _, err := h.ChangelogsService.Update(ctx, req.TeamID, req.ChangelogID, req.Name); err != nil {
		c.Log.WithError(err).Error("failed to update changelog")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, req)
}

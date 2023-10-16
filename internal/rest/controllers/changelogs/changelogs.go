package changelogs

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetChangelogsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	Offset *int `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit  *int `query:"limit" validate:"numeric,gt=0" default:"10"`
}

type GetChangelogsResponse struct {
	Changelogs []GetChangelogsResponseChangelog `json:"changelogs"`
	TotalCount int                              `json:"totalCount"`
}

type GetChangelogsResponseChangelog struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelogs(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetChangelogsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetChangelogsRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

type PostChangelogsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

type PostChangelogsResponse struct {
	Name string `json:"name"`
}

func (h *Handlers) PostChangelogs(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostChangelogsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostChangelogsRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

type GetChangelogRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
}

type GetChangelogResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelog(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetChangelogRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetChangelogRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

type DeleteChangelogRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteChangelog(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteChangelogRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteChangelogRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

type PutChangelogRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint   `param:"changelogId" validate:"required,numeric,gte=0"`
	Name        string `json:"name" validate:"required,max=255"`
}

func (h *Handlers) PutChangelog(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutChangelogRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutChangelogRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

package changelogs

import (
	"time"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
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
	return echo.ErrNotFound // TODO: implement
}

type PostChangelogsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

type PostChangelogsResponse struct {
	Name string `json:"name"`
}

func (h *Handlers) PostChangelogs(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type GetChangelogRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID string `param:"changelogId" validate:"required,uuid"`
}

type GetChangelogResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelog(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type DeleteChangelogRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID string `param:"changelogId" validate:"required,uuid"`
}

func (h *Handlers) DeleteChangelog(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type PutChangelogRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID string `param:"changelogId" validate:"required,uuid"`
	Name        string `json:"name"`
}

func (h *Handlers) PutChangelog(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

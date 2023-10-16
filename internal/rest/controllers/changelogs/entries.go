package changelogs

import (
	"time"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
)

type GetChangelogEntriesRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID string `param:"changelogId" validate:"required,uuid"`
	Offset      *int   `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit       *int   `query:"limit" validate:"numeric,gt=0" default:"10"`
}

type GetChangelogEntriesResponse struct {
	Entries    []GetChangelogEntriesResponseEntry `json:"entries"`
	TotalCount int                                `json:"totalCount"`
}

type GetChangelogEntriesResponseEntry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelogEntries(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type PostChangelogEntriesRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID string `param:"changelogId" validate:"required,uuid"`
}

func (h *Handlers) PostChangelogEntries(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type GetChangelogEntryRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID string `param:"changelogId" validate:"required,uuid"`
	EntryID     string `param:"entryId" validate:"required,uuid"`
}

type GetChangelogEntryResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelogEntry(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type DeleteChangelogEntryRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID string `param:"changelogId" validate:"required,uuid"`
	EntryID     string `param:"entryId" validate:"required,uuid"`
}

func (h *Handlers) DeleteChangelogEntry(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type PutChangelogEntryRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID string `param:"changelogId" validate:"required,uuid"`
	EntryID     string `param:"entryId" validate:"required,uuid"`
	Title       string `json:"title"`
}

func (h *Handlers) PutChangelogEntry(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

package changelogs

import (
	"time"

	"github.com/labstack/echo/v4"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
)

type GetChangelogEntriesRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
	Offset      *int `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit       *int `query:"limit" validate:"numeric,gt=0" default:"10"`
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
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) PostChangelogEntries(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type GetChangelogEntryRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
	EntryID     uint `param:"entryId" validate:"required,numeric,gte=0"`
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
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
	EntryID     uint `param:"entryId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteChangelogEntry(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

type PutChangelogEntryRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint   `param:"changelogId" validate:"required,numeric,gte=0"`
	EntryID     uint   `param:"entryId" validate:"required,numeric,gte=0"`
	Title       string `json:"title" validate:"required,max=512"`
}

func (h *Handlers) PutChangelogEntry(c hs.AuthenticatedContext) error {
	return echo.ErrNotFound // TODO: implement
}

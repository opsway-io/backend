package changelogs

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetChangelogEntriesRequest struct {
	TeamID      uint    `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint    `param:"changelogId" validate:"required,numeric,gte=0"`
	Offset      *int    `query:"offset" validate:"numeric,gte=0" default:"0"`
	Limit       *int    `query:"limit" validate:"numeric,gt=0" default:"10"`
	Query       *string `query:"query" validate:"max=255"`
}

type GetChangelogEntriesResponse struct {
	Entries    []GetChangelogEntriesResponseEntry `json:"entries"`
	TotalCount int                                `json:"totalCount"`
}

type GetChangelogEntriesResponseEntry struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelogEntries(c hs.AuthenticatedContext) error {
	ctx := c.Request().Context()

	req, err := helpers.Bind[GetChangelogEntriesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetChangelogEntriesRequest")

		return echo.ErrBadRequest
	}

	entries, totalCount, err := h.ChangelogsService.GetEntriesWithAuthors(ctx, req.TeamID, req.ChangelogID, req.Offset, req.Limit, req.Query)
	if err != nil {
		c.Log.WithError(err).Error("failed to get changelog entries")

		return echo.ErrInternalServerError
	}

	resp := h.newGetChangelogEntriesResponse(entries, totalCount)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetChangelogEntriesResponse(entries []entities.ChangelogEntry, totalCount int) GetChangelogEntriesResponse {
	response := make([]GetChangelogEntriesResponseEntry, len(entries))

	for i, entry := range entries {
		response[i] = GetChangelogEntriesResponseEntry{
			ID:        entry.ID,
			Title:     entry.Title,
			CreatedAt: entry.CreatedAt,
			UpdatedAt: entry.UpdatedAt,
		}
	}

	return GetChangelogEntriesResponse{
		Entries:    response,
		TotalCount: totalCount,
	}
}

type PostChangelogEntriesRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) PostChangelogEntries(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostChangelogEntriesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostChangelogEntriesRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

type GetChangelogEntryRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
	EntryID     uint `param:"entryId" validate:"required,numeric,gte=0"`
}

type GetChangelogEntryResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (h *Handlers) GetChangelogEntry(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetChangelogEntryRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetChangelogEntryRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

type DeleteChangelogEntryRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint `param:"changelogId" validate:"required,numeric,gte=0"`
	EntryID     uint `param:"entryId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteChangelogEntry(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteChangelogEntryRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteChangelogEntryRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

type PutChangelogEntryRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	ChangelogID uint   `param:"changelogId" validate:"required,numeric,gte=0"`
	EntryID     uint   `param:"entryId" validate:"required,numeric,gte=0"`
	Title       string `json:"title" validate:"required,max=512"`
}

func (h *Handlers) PutChangelogEntry(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutChangelogEntryRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutChangelogEntryRequest")

		return echo.ErrBadRequest
	}

	// TODO: implement

	return c.JSON(http.StatusOK, req)
}

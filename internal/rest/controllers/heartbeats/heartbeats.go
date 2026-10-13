package heartbeats

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/heartbeat"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type HeartbeatResponse struct {
	ID        uint                     `json:"id"`
	Name      string                   `json:"name"`
	Status    entities.HeartbeatStatus `json:"status"`
	Interval  uint                     `json:"interval"`
	Grace     uint                     `json:"grace"`
	LastPing  *time.Time               `json:"lastPing"`
	CreatedAt time.Time                `json:"createdAt"`
	UpdatedAt time.Time                `json:"updatedAt"`
}

func newHeartbeatResponse(h *entities.Heartbeat, now time.Time) HeartbeatResponse {
	return HeartbeatResponse{
		ID:        h.ID,
		Name:      h.Name,
		Status:    h.StatusAt(now),
		Interval:  h.Interval,
		Grace:     h.Grace,
		LastPing:  h.LastPingAt,
		CreatedAt: h.CreatedAt,
		UpdatedAt: h.UpdatedAt,
	}
}

type GetHeartbeatsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetHeartbeats(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetHeartbeatsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetHeartbeatsRequest")

		return echo.ErrBadRequest
	}

	heartbeats, err := h.HeartbeatService.GetByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get heartbeats")

		return echo.ErrInternalServerError
	}

	now := time.Now()
	resp := make([]HeartbeatResponse, len(*heartbeats))
	for i := range *heartbeats {
		resp[i] = newHeartbeatResponse(&(*heartbeats)[i], now)
	}

	return c.JSON(http.StatusOK, resp)
}

type HeartbeatIDRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	HeartbeatID uint `param:"heartbeatId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetHeartbeat(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[HeartbeatIDRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind HeartbeatIDRequest")

		return echo.ErrBadRequest
	}

	hb, err := h.HeartbeatService.GetByTeamIDAndID(c.Request().Context(), req.TeamID, req.HeartbeatID)
	if err != nil {
		return h.serviceError(c, err)
	}

	return c.JSON(http.StatusOK, newHeartbeatResponse(hb, time.Now()))
}

// Interval and grace are in minutes. Interval max is 31 days, grace max 1 day.
type PostHeartbeatRequest struct {
	TeamID   uint   `param:"teamId" validate:"required,numeric,gte=0"`
	Name     string `json:"name" validate:"required,min=1,max=255"`
	Interval uint   `json:"interval" validate:"required,gte=1,lte=44640"`
	Grace    uint   `json:"grace" validate:"gte=0,lte=1440"`
}

func (h *Handlers) PostHeartbeat(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostHeartbeatRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostHeartbeatRequest")

		return echo.ErrBadRequest
	}

	hb := &entities.Heartbeat{
		TeamID:   req.TeamID,
		Name:     strings.TrimSpace(req.Name),
		Interval: req.Interval,
		Grace:    req.Grace,
	}

	if err := h.HeartbeatService.Create(c.Request().Context(), hb); err != nil {
		c.Log.WithError(err).Error("failed to create heartbeat")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, newHeartbeatResponse(hb, time.Now()))
}

type PutHeartbeatRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	HeartbeatID uint   `param:"heartbeatId" validate:"required,numeric,gte=0"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Interval    uint   `json:"interval" validate:"required,gte=1,lte=44640"`
	Grace       uint   `json:"grace" validate:"gte=0,lte=1440"`
}

func (h *Handlers) PutHeartbeat(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutHeartbeatRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutHeartbeatRequest")

		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()

	if err := h.HeartbeatService.Update(ctx, &entities.Heartbeat{
		ID:       req.HeartbeatID,
		TeamID:   req.TeamID,
		Name:     strings.TrimSpace(req.Name),
		Interval: req.Interval,
		Grace:    req.Grace,
	}); err != nil {
		return h.serviceError(c, err)
	}

	hb, err := h.HeartbeatService.GetByTeamIDAndID(ctx, req.TeamID, req.HeartbeatID)
	if err != nil {
		return h.serviceError(c, err)
	}

	return c.JSON(http.StatusOK, newHeartbeatResponse(hb, time.Now()))
}

func (h *Handlers) DeleteHeartbeat(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[HeartbeatIDRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind HeartbeatIDRequest")

		return echo.ErrBadRequest
	}

	if err := h.HeartbeatService.Delete(c.Request().Context(), req.TeamID, req.HeartbeatID); err != nil {
		return h.serviceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// PingHeartbeat is called (unauthenticated) by cron jobs / background tasks.
func (h *Handlers) PingHeartbeat(c echo.Context) error {
	req, err := helpers.Bind[HeartbeatIDRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	// The endpoint is unauthenticated, so don't echo heartbeat details back.
	if _, err := h.HeartbeatService.Ping(c.Request().Context(), req.TeamID, req.HeartbeatID); err != nil {
		if errors.Is(err, heartbeat.ErrNotFound) {
			return echo.ErrNotFound
		}

		h.Logger.WithError(err).Error("failed to ping heartbeat")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handlers) serviceError(c hs.AuthenticatedContext, err error) error {
	if errors.Is(err, heartbeat.ErrNotFound) {
		return echo.ErrNotFound
	}

	c.Log.WithError(err).Error("heartbeat operation failed")

	return echo.ErrInternalServerError
}

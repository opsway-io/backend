package heartbeats

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/heartbeats"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	TeamService      team.Service
	HeartbeatService heartbeats.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	heartbeatService heartbeats.Service,
) {
	h := &Handlers{
		TeamService:      teamService,
		HeartbeatService: heartbeatService,
	}

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	cg := e.Group("/teams/:teamId/heartbeats")

	cg.GET("", AuthHandler(h.GetHeartbeats))
	cg.POST("", AuthHandler(h.PostHeartbeat))
	cg.GET("/:heartbeatId", AuthHandler(h.GetHeartbeat))
	cg.PUT("/:heartbeatId", AuthHandler(h.PutHeartbeat))
	cg.DELETE("/:heartbeatId", AuthHandler(h.DeleteHeartbeat))
	cg.POST("/:heartbeatId/ping", AuthHandler(h.PingHeartbeat))
}

type GetHeartbeatsRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetHeartbeats(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetHeartbeatsRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetHeartbeatsRequest")
		return echo.ErrBadRequest
	}

	hbs, err := h.HeartbeatService.GetAllByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get heartbeats")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, hbs)
}

type GetHeartbeatRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	HeartbeatID uint `param:"heartbeatId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetHeartbeat(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetHeartbeatRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetHeartbeatRequest")
		return echo.ErrBadRequest
	}

	hb, err := h.HeartbeatService.GetByIDAndTeamID(c.Request().Context(), req.TeamID, req.HeartbeatID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get heartbeat")
		return echo.ErrNotFound
	}

	return c.JSON(http.StatusOK, hb)
}

type PostHeartbeatRequest struct {
	TeamID   uint   `param:"teamId" validate:"required,numeric,gte=0"`
	Name     string `json:"name" validate:"required"`
	Interval int64  `json:"interval" validate:"required,numeric,gte=1"`
	Grace    int64  `json:"grace" validate:"required,numeric,gte=0"`
}

func (h *Handlers) PostHeartbeat(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PostHeartbeatRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostHeartbeatRequest")
		return echo.ErrBadRequest
	}

	hb := &entities.Heartbeat{
		TeamID:   req.TeamID,
		Name:     req.Name,
		Status:   entities.HeartbeatStatusPaused,
		Interval: time.Duration(req.Interval) * time.Minute,
		Grace:    time.Duration(req.Grace) * time.Minute,
	}

	err = h.HeartbeatService.Create(c.Request().Context(), hb)
	if err != nil {
		c.Log.WithError(err).Error("failed to create heartbeat")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, hb)
}

type PutHeartbeatRequest struct {
	TeamID      uint   `param:"teamId" validate:"required,numeric,gte=0"`
	HeartbeatID uint   `param:"heartbeatId" validate:"required,numeric,gte=0"`
	Name        string `json:"name" validate:"required"`
	Interval    int64  `json:"interval" validate:"required,numeric,gte=1"`
	Grace       int64  `json:"grace" validate:"required,numeric,gte=0"`
}

func (h *Handlers) PutHeartbeat(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PutHeartbeatRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutHeartbeatRequest")
		return echo.ErrBadRequest
	}

	hb, err := h.HeartbeatService.GetByIDAndTeamID(c.Request().Context(), req.TeamID, req.HeartbeatID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get heartbeat for update")
		return echo.ErrNotFound
	}

	hb.Name = req.Name
	hb.Interval = time.Duration(req.Interval) * time.Minute
	hb.Grace = time.Duration(req.Grace) * time.Minute

	err = h.HeartbeatService.Update(c.Request().Context(), hb)
	if err != nil {
		c.Log.WithError(err).Error("failed to update heartbeat")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, hb)
}

type DeleteHeartbeatRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	HeartbeatID uint `param:"heartbeatId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteHeartbeat(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteHeartbeatRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteHeartbeatRequest")
		return echo.ErrBadRequest
	}

	hb, err := h.HeartbeatService.GetByIDAndTeamID(c.Request().Context(), req.TeamID, req.HeartbeatID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get heartbeat for deletion")
		return echo.ErrNotFound
	}

	err = h.HeartbeatService.Delete(c.Request().Context(), hb)
	if err != nil {
		c.Log.WithError(err).Error("failed to delete heartbeat")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

type PingHeartbeatRequest struct {
	TeamID      uint `param:"teamId" validate:"required,numeric,gte=0"`
	HeartbeatID uint `param:"heartbeatId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) PingHeartbeat(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PingHeartbeatRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PingHeartbeatRequest")
		return echo.ErrBadRequest
	}

	err = h.HeartbeatService.Ping(c.Request().Context(), req.HeartbeatID)
	if err != nil {
		c.Log.WithError(err).Error("failed to ping heartbeat")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}

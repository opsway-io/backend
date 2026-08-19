package statuspages

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/rest/middleware"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/team"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type Handlers struct {
	TeamService       team.Service
	StatusPageService statuspage.Service
}

func Register(
	e *echo.Group,
	logger *logrus.Entry,
	teamService team.Service,
	statusPageService statuspage.Service,
) {
	h := &Handlers{
		TeamService:       teamService,
		StatusPageService: statusPageService,
	}

	TeamGuard := middleware.TeamGuardFactory(logger, teamService)
	AllowedRoles := middleware.RoleGuardFactory(logger, teamService)

	AuthHandler := handlers.AuthenticatedHandlerFactory(logger)

	statusPagesGroup := e.Group(
		"/teams/:teamId/status-pages",
		TeamGuard(),
	)

	statusPagesGroup.GET("", AuthHandler(h.GetStatusPages))
	statusPagesGroup.POST("", AuthHandler(h.PostStatusPage), AllowedRoles(middleware.UserRoleOwner, middleware.UserRoleAdmin))
	statusPagesGroup.GET("/:statusPageId", AuthHandler(h.GetStatusPage))
	statusPagesGroup.PUT("/:statusPageId", AuthHandler(h.PutStatusPage), AllowedRoles(middleware.UserRoleOwner, middleware.UserRoleAdmin))
	statusPagesGroup.DELETE("/:statusPageId", AuthHandler(h.DeleteStatusPage), AllowedRoles(middleware.UserRoleOwner, middleware.UserRoleAdmin))
}

type GetStatusPagesRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetStatusPagesResponse struct {
	StatusPages []GetStatusPageResponse `json:"statusPages"`
}

type GetStatusPageResponse struct {
	ID           uint     `json:"id"`
	Name         string   `json:"name"`
	Domain       string   `json:"domain"`
	LogoURL      string   `json:"logoUrl"`
	LogoLink     string   `json:"logoLink"`
	FaviconURL   string   `json:"faviconUrl"`
	Layout       string   `json:"layout"`
	CustomCSS            string   `json:"customCss"`
	HeaderHTML           string   `json:"headerHtml"`
	FooterHTML           string   `json:"footerHtml"`
	CustomComponentsHTML string   `json:"customComponentsHtml"`
	ShowBranding         bool     `json:"showBranding"`
	IsPrivate            bool     `json:"isPrivate"`
	MonitorIDs   []uint   `json:"monitorIds"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

func (h *Handlers) GetStatusPages(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetStatusPagesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetStatusPagesRequest")
		return echo.ErrBadRequest
	}

	statusPages, err := h.StatusPageService.GetByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get status pages")
		return echo.ErrInternalServerError
	}

	res := GetStatusPagesResponse{
		StatusPages: make([]GetStatusPageResponse, len(statusPages)),
	}

	for i, sp := range statusPages {
		var monitorIDs []uint
		for _, m := range sp.Monitors {
			monitorIDs = append(monitorIDs, m.ID)
		}

		res.StatusPages[i] = GetStatusPageResponse{
			ID:           sp.ID,
			Name:         sp.Name,
			Domain:       sp.Domain,
			LogoURL:      sp.LogoURL,
			LogoLink:     sp.LogoLink,
			FaviconURL:   sp.FaviconURL,
			Layout:       sp.Layout,
			CustomCSS:            sp.CustomCSS,
			HeaderHTML:           sp.HeaderHTML,
			FooterHTML:           sp.FooterHTML,
			CustomComponentsHTML: sp.CustomComponentsHTML,
			ShowBranding:         sp.ShowBranding,
			IsPrivate:            sp.IsPrivate,
			MonitorIDs:   monitorIDs,
			CreatedAt:    sp.CreatedAt.String(),
			UpdatedAt:    sp.UpdatedAt.String(),
		}
	}

	return c.JSON(http.StatusOK, res)
}

type PostStatusPageRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gt=0"`
	Name   string `json:"name" validate:"required,max=255"`
	Domain string `json:"domain" validate:"required,max=255"`
}

func (h *Handlers) PostStatusPage(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PostStatusPageRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostStatusPageRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()
	teamEntity, err := h.TeamService.GetByID(ctx, req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get team")
		return echo.ErrInternalServerError
	}

	limit := teamEntity.GetStatusPageLimit()
	if limit != -1 {
		count, err := h.StatusPageService.CountByTeamID(ctx, req.TeamID)
		if err != nil {
			c.Log.WithError(err).Debug("failed to get status page count")
			return echo.ErrInternalServerError
		}
		if count >= int64(limit) {
			return echo.NewHTTPError(http.StatusPaymentRequired, "Payment Required")
		}
	}

	sp := &entities.StatusPage{
		TeamID: req.TeamID,
		Name:   req.Name,
		Domain: req.Domain,
	}

	if err := h.StatusPageService.Create(c.Request().Context(), sp); err != nil {
		c.Log.WithError(err).Error("failed to create status page")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, GetStatusPageResponse{
		ID:           sp.ID,
		Name:         sp.Name,
		Domain:       sp.Domain,
		ShowBranding: sp.ShowBranding,
		IsPrivate:    sp.IsPrivate,
		CreatedAt:    sp.CreatedAt.String(),
		UpdatedAt:    sp.UpdatedAt.String(),
	})
}

type GetStatusPageRequest struct {
	TeamID       uint `param:"teamId" validate:"required,numeric,gt=0"`
	StatusPageID uint `param:"statusPageId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) GetStatusPage(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[GetStatusPageRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetStatusPageRequest")
		return echo.ErrBadRequest
	}

	sp, err := h.StatusPageService.GetByIDAndTeamID(c.Request().Context(), req.StatusPageID, req.TeamID)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		c.Log.WithError(err).Error("failed to get status page")
		return echo.ErrInternalServerError
	}

	var monitorIDs []uint
	for _, m := range sp.Monitors {
		monitorIDs = append(monitorIDs, m.ID)
	}

	return c.JSON(http.StatusOK, GetStatusPageResponse{
		ID:           sp.ID,
		Name:         sp.Name,
		Domain:       sp.Domain,
		LogoURL:      sp.LogoURL,
		LogoLink:     sp.LogoLink,
		FaviconURL:   sp.FaviconURL,
		Layout:       sp.Layout,
		CustomCSS:            sp.CustomCSS,
		HeaderHTML:           sp.HeaderHTML,
		FooterHTML:           sp.FooterHTML,
		CustomComponentsHTML: sp.CustomComponentsHTML,
		ShowBranding:         sp.ShowBranding,
		IsPrivate:            sp.IsPrivate,
		MonitorIDs:   monitorIDs,
		CreatedAt:    sp.CreatedAt.String(),
		UpdatedAt:    sp.UpdatedAt.String(),
	})
}

type PutStatusPageRequest struct {
	TeamID       uint    `param:"teamId" validate:"required,numeric,gt=0"`
	StatusPageID uint    `param:"statusPageId" validate:"required,numeric,gt=0"`
	Name         string  `json:"name" validate:"required,max=255"`
	Domain       string  `json:"domain" validate:"required,max=255"`
	LogoURL      string  `json:"logoUrl"`
	LogoLink     string  `json:"logoLink"`
	FaviconURL   string  `json:"faviconUrl"`
	Layout       string  `json:"layout"`
	CustomCSS            string  `json:"customCss"`
	HeaderHTML           string  `json:"headerHtml"`
	FooterHTML           string  `json:"footerHtml"`
	CustomComponentsHTML string  `json:"customComponentsHtml"`
	ShowBranding         *bool   `json:"showBranding"`
	IsPrivate            *bool   `json:"isPrivate"`
	Password             string  `json:"password"`
	MonitorIDs   []uint  `json:"monitorIds"`
}

func (h *Handlers) PutStatusPage(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[PutStatusPageRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutStatusPageRequest")
		return echo.ErrBadRequest
	}

	sp, err := h.StatusPageService.GetByIDAndTeamID(c.Request().Context(), req.StatusPageID, req.TeamID)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		c.Log.WithError(err).Error("failed to get status page")
		return echo.ErrInternalServerError
	}

	sp.Name = req.Name
	sp.Domain = req.Domain
	sp.LogoURL = req.LogoURL
	sp.LogoLink = req.LogoLink
	sp.FaviconURL = req.FaviconURL
	sp.CustomCSS = req.CustomCSS
	sp.HeaderHTML = req.HeaderHTML
	sp.FooterHTML = req.FooterHTML
	sp.CustomComponentsHTML = req.CustomComponentsHTML
	if req.ShowBranding != nil {
		sp.ShowBranding = *req.ShowBranding
	}
	if req.IsPrivate != nil {
		sp.IsPrivate = *req.IsPrivate
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.Log.WithError(err).Error("failed to hash status page password")
			return echo.ErrInternalServerError
		}
		sp.PasswordHash = string(hash)
	} else if req.IsPrivate != nil && !*req.IsPrivate {
		sp.PasswordHash = ""
	}

	if req.Layout != "" {
		sp.Layout = req.Layout
	} else {
		sp.Layout = "STATS"
	}

	if err := h.StatusPageService.Update(c.Request().Context(), sp); err != nil {
		c.Log.WithError(err).Error("failed to update status page")
		return echo.ErrInternalServerError
	}

	if req.MonitorIDs != nil {
		var monitors []entities.Monitor
		for _, mID := range req.MonitorIDs {
			monitors = append(monitors, entities.Monitor{ID: mID})
		}
		if err := h.StatusPageService.ReplaceMonitors(c.Request().Context(), sp, monitors); err != nil {
			c.Log.WithError(err).Error("failed to update status page monitors")
			return echo.ErrInternalServerError
		}
	}

	// Refetch to get updated relations
	sp, _ = h.StatusPageService.GetByIDAndTeamID(c.Request().Context(), req.StatusPageID, req.TeamID)

	var monitorIDs []uint
	for _, m := range sp.Monitors {
		monitorIDs = append(monitorIDs, m.ID)
	}

	return c.JSON(http.StatusOK, GetStatusPageResponse{
		ID:           sp.ID,
		Name:         sp.Name,
		Domain:       sp.Domain,
		LogoURL:      sp.LogoURL,
		LogoLink:     sp.LogoLink,
		FaviconURL:   sp.FaviconURL,
		Layout:       sp.Layout,
		CustomCSS:            sp.CustomCSS,
		HeaderHTML:           sp.HeaderHTML,
		FooterHTML:           sp.FooterHTML,
		CustomComponentsHTML: sp.CustomComponentsHTML,
		ShowBranding:         sp.ShowBranding,
		IsPrivate:            sp.IsPrivate,
		MonitorIDs:   monitorIDs,
		CreatedAt:    sp.CreatedAt.String(),
		UpdatedAt:    sp.UpdatedAt.String(),
	})
}

type DeleteStatusPageRequest struct {
	TeamID       uint `param:"teamId" validate:"required,numeric,gt=0"`
	StatusPageID uint `param:"statusPageId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteStatusPage(c handlers.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteStatusPageRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteStatusPageRequest")
		return echo.ErrBadRequest
	}

	if err := h.StatusPageService.Delete(c.Request().Context(), req.StatusPageID, req.TeamID); err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		c.Log.WithError(err).Error("failed to delete status page")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

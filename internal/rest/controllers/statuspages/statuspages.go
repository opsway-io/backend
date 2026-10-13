package statuspages

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/statuspage"
)

type StatusPageGroup struct {
	ID         uint   `json:"id,omitempty"`
	Name       string `json:"name" validate:"required,min=1,max=255"`
	Order      int    `json:"order" validate:"gte=0"`
	MonitorIDs []uint `json:"monitorIds" validate:"max=500,dive,gt=0"`
}

type StatusPageResponse struct {
	ID                   uint              `json:"id"`
	Name                 string            `json:"name"`
	Domain               string            `json:"domain"`
	LogoURL              string            `json:"logoUrl"`
	LogoLink             string            `json:"logoLink"`
	FaviconURL           string            `json:"faviconUrl"`
	Layout               string            `json:"layout"`
	CustomCSS            string            `json:"customCss"`
	HeaderHTML           string            `json:"headerHtml"`
	FooterHTML           string            `json:"footerHtml"`
	CustomComponentsHTML string            `json:"customComponentsHtml"`
	ShowBranding         bool              `json:"showBranding"`
	IsPrivate            bool              `json:"isPrivate"`
	SupportURL           string            `json:"supportUrl"`
	MonitorIDs           []uint            `json:"monitorIds"`
	Groups               []StatusPageGroup `json:"groups"`
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
}

func toUints(a pq.Int64Array) []uint {
	out := make([]uint, len(a))
	for i, v := range a {
		out[i] = uint(v)
	}

	return out
}

func toInt64Array(a []uint) pq.Int64Array {
	out := make(pq.Int64Array, 0, len(a))
	seen := map[uint]struct{}{}
	for _, v := range a {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, int64(v))
	}

	return out
}

func newStatusPageResponse(p *entities.StatusPage) StatusPageResponse {
	groups := make([]StatusPageGroup, len(p.Groups))
	for i, g := range p.Groups {
		groups[i] = StatusPageGroup{ID: g.ID, Name: g.Name, Order: g.Order, MonitorIDs: toUints(g.MonitorIDs)}
	}

	return StatusPageResponse{
		ID:                   p.ID,
		Name:                 p.Name,
		Domain:               p.Domain,
		LogoURL:              p.LogoURL,
		LogoLink:             p.LogoLink,
		FaviconURL:           p.FaviconURL,
		Layout:               p.Layout,
		CustomCSS:            p.CustomCSS,
		HeaderHTML:           p.HeaderHTML,
		FooterHTML:           p.FooterHTML,
		CustomComponentsHTML: p.CustomComponentsHTML,
		ShowBranding:         p.ShowBranding,
		IsPrivate:            p.IsPrivate,
		SupportURL:           p.SupportURL,
		MonitorIDs:           toUints(p.MonitorIDs),
		Groups:               groups,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

type GetStatusPagesRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

type GetStatusPagesResponse struct {
	StatusPages []StatusPageResponse `json:"statusPages"`
}

func (h *Handlers) GetStatusPages(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetStatusPagesRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetStatusPagesRequest")

		return echo.ErrBadRequest
	}

	pages, err := h.StatusPageService.GetByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get status pages")

		return echo.ErrInternalServerError
	}

	resp := GetStatusPagesResponse{StatusPages: make([]StatusPageResponse, len(*pages))}
	for i := range *pages {
		resp.StatusPages[i] = newStatusPageResponse(&(*pages)[i])
	}

	return c.JSON(http.StatusOK, resp)
}

type StatusPageIDRequest struct {
	TeamID       uint `param:"teamId" validate:"required,numeric,gte=0"`
	StatusPageID uint `param:"statusPageId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) GetStatusPage(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[StatusPageIDRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind StatusPageIDRequest")

		return echo.ErrBadRequest
	}

	page, err := h.StatusPageService.GetByTeamIDAndID(c.Request().Context(), req.TeamID, req.StatusPageID)
	if err != nil {
		return serviceError(c, err)
	}

	return c.JSON(http.StatusOK, newStatusPageResponse(page))
}

type PostStatusPageRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gte=0"`
	Name   string `json:"name" validate:"required,min=1,max=255"`
	Domain string `json:"domain" validate:"required,fqdn,max=253"`
}

func (h *Handlers) PostStatusPage(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostStatusPageRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostStatusPageRequest")

		return echo.ErrBadRequest
	}

	page := &entities.StatusPage{
		TeamID:       req.TeamID,
		Name:         strings.TrimSpace(req.Name),
		Domain:       req.Domain,
		Layout:       string(entities.StatusPageLayoutStats),
		ShowBranding: true,
	}

	if err := h.StatusPageService.Create(c.Request().Context(), page); err != nil {
		return serviceError(c, err)
	}

	return c.JSON(http.StatusCreated, newStatusPageResponse(page))
}

type PutStatusPageRequest struct {
	TeamID               uint              `param:"teamId" validate:"required,numeric,gte=0"`
	StatusPageID         uint              `param:"statusPageId" validate:"required,numeric,gte=0"`
	Name                 string            `json:"name" validate:"required,min=1,max=255"`
	Domain               string            `json:"domain" validate:"required,fqdn,max=253"`
	LogoURL              string            `json:"logoUrl" validate:"omitempty,url,max=2048"`
	LogoLink             string            `json:"logoLink" validate:"omitempty,url,max=2048"`
	FaviconURL           string            `json:"faviconUrl" validate:"omitempty,url,max=2048"`
	Layout               string            `json:"layout" validate:"omitempty,oneof=STATS COMPACT SIMPLE"`
	CustomCSS            string            `json:"customCss" validate:"max=100000"`
	HeaderHTML           string            `json:"headerHtml" validate:"max=100000"`
	FooterHTML           string            `json:"footerHtml" validate:"max=100000"`
	CustomComponentsHTML string            `json:"customComponentsHtml" validate:"max=100000"`
	ShowBranding         *bool             `json:"showBranding"`
	IsPrivate            bool              `json:"isPrivate"`
	SupportURL           string            `json:"supportUrl" validate:"max=2048"`
	Password             string            `json:"password" validate:"omitempty,min=4,max=72"`
	MonitorIDs           []uint            `json:"monitorIds" validate:"max=500,dive,gt=0"`
	Groups               []StatusPageGroup `json:"groups" validate:"max=50,dive"`
}

func (h *Handlers) PutStatusPage(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutStatusPageRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PutStatusPageRequest")

		return echo.ErrBadRequest
	}

	supportURL := strings.TrimSpace(req.SupportURL)
	if supportURL != "" && !isValidSupportURL(supportURL) {
		return echo.NewHTTPError(http.StatusBadRequest, "supportUrl must be an http(s) or mailto: URL")
	}

	for _, u := range []string{req.LogoURL, req.LogoLink, req.FaviconURL} {
		if u != "" && !isHTTPURL(u) {
			return echo.NewHTTPError(http.StatusBadRequest, "logoUrl, logoLink and faviconUrl must be http(s) URLs")
		}
	}

	layout := req.Layout
	if layout == "" {
		layout = string(entities.StatusPageLayoutStats)
	}

	showBranding := true
	if req.ShowBranding != nil {
		showBranding = *req.ShowBranding
	}

	page := &entities.StatusPage{
		ID:                   req.StatusPageID,
		TeamID:               req.TeamID,
		Name:                 strings.TrimSpace(req.Name),
		Domain:               req.Domain,
		LogoURL:              req.LogoURL,
		LogoLink:             req.LogoLink,
		FaviconURL:           req.FaviconURL,
		Layout:               layout,
		CustomCSS:            req.CustomCSS,
		HeaderHTML:           req.HeaderHTML,
		FooterHTML:           req.FooterHTML,
		CustomComponentsHTML: req.CustomComponentsHTML,
		ShowBranding:         showBranding,
		IsPrivate:            req.IsPrivate,
		SupportURL:           supportURL,
		MonitorIDs:           toInt64Array(req.MonitorIDs),
	}

	for i, g := range req.Groups {
		order := g.Order
		if order == 0 {
			order = i
		}
		page.Groups = append(page.Groups, entities.StatusPageGroup{
			Name:       strings.TrimSpace(g.Name),
			Order:      order,
			MonitorIDs: toInt64Array(g.MonitorIDs),
		})
	}

	if req.IsPrivate && req.Password != "" {
		if err := page.SetPassword(req.Password); err != nil {
			c.Log.WithError(err).Error("failed to hash status page password")

			return echo.ErrInternalServerError
		}
	}

	ctx := c.Request().Context()

	if err := h.StatusPageService.Update(ctx, page); err != nil {
		return serviceError(c, err)
	}

	updated, err := h.StatusPageService.GetByTeamIDAndID(ctx, req.TeamID, req.StatusPageID)
	if err != nil {
		return serviceError(c, err)
	}

	return c.JSON(http.StatusOK, newStatusPageResponse(updated))
}

func (h *Handlers) DeleteStatusPage(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[StatusPageIDRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind StatusPageIDRequest")

		return echo.ErrBadRequest
	}

	if err := h.StatusPageService.Delete(c.Request().Context(), req.TeamID, req.StatusPageID); err != nil {
		return serviceError(c, err)
	}

	return c.NoContent(http.StatusNoContent)
}

func isHTTPURL(u string) bool {
	lower := strings.ToLower(u)

	return strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://")
}

func isValidSupportURL(u string) bool {
	lower := strings.ToLower(u)

	return strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "mailto:")
}

func serviceError(c hs.AuthenticatedContext, err error) error {
	switch {
	case errors.Is(err, statuspage.ErrNotFound):
		return echo.ErrNotFound
	case errors.Is(err, statuspage.ErrDomainTaken):
		return echo.NewHTTPError(http.StatusConflict, "domain is already in use by another status page")
	case errors.Is(err, statuspage.ErrInvalidMonitor):
		return echo.NewHTTPError(http.StatusBadRequest, "one or more monitors do not exist")
	case errors.Is(err, statuspage.ErrPasswordRequired):
		return echo.NewHTTPError(http.StatusBadRequest, "a password is required for private status pages")
	default:
		c.Log.WithError(err).Error("status page operation failed")

		return echo.ErrInternalServerError
	}
}

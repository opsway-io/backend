package statuspages

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/statuspage"
)

const passwordHeader = "X-Status-Page-Password"

type GetPublicStatusPageResponse struct {
	Name                 string `json:"name"`
	LogoURL              string `json:"logoUrl"`
	LogoLink             string `json:"logoLink"`
	FaviconURL           string `json:"faviconUrl"`
	Layout               string `json:"layout"`
	CustomCSS            string `json:"customCss"`
	HeaderHTML           string `json:"headerHtml"`
	FooterHTML           string `json:"footerHtml"`
	CustomComponentsHTML string `json:"customComponentsHtml"`
	ShowBranding         bool   `json:"showBranding"`
	IsPrivate            bool   `json:"isPrivate"`
	SupportURL           string `json:"supportUrl,omitempty"`
	*statuspage.PublicView
}

// loadPage resolves the status page from the :domain param and enforces the password of private pages.
func (h *Handlers) loadPage(c echo.Context, requirePassword bool) (*entities.StatusPage, error) {
	domain := c.Param("domain")
	if domain == "" || len(domain) > 253 {
		return nil, echo.ErrNotFound
	}

	page, err := h.StatusPageService.GetByDomain(c.Request().Context(), domain)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return nil, echo.ErrNotFound
		}

		h.Logger.WithError(err).Error("failed to get status page by domain")

		return nil, echo.ErrInternalServerError
	}

	if requirePassword && page.IsPrivate && !page.CheckPassword(c.Request().Header.Get(passwordHeader)) {
		return nil, echo.ErrUnauthorized
	}

	return page, nil
}

func (h *Handlers) GetPublicStatusPage(c echo.Context) error {
	page, err := h.loadPage(c, true)
	if err != nil {
		return err
	}

	view, err := h.StatusPageService.GetPublicView(c.Request().Context(), page)
	if err != nil {
		h.Logger.WithError(err).Error("failed to build public status page")

		return echo.ErrInternalServerError
	}

	c.Response().Header().Set("Cache-Control", "no-store")

	return c.JSON(http.StatusOK, GetPublicStatusPageResponse{
		Name:                 page.Name,
		LogoURL:              page.LogoURL,
		LogoLink:             page.LogoLink,
		FaviconURL:           page.FaviconURL,
		Layout:               page.Layout,
		CustomCSS:            page.CustomCSS,
		HeaderHTML:           page.HeaderHTML,
		FooterHTML:           page.FooterHTML,
		CustomComponentsHTML: page.CustomComponentsHTML,
		ShowBranding:         page.ShowBranding,
		IsPrivate:            page.IsPrivate,
		SupportURL:           page.SupportURL,
		PublicView:           view,
	})
}

type PostSubscribeRequest struct {
	Email       string `json:"email"`
	Channel     string `json:"channel"`
	Destination string `json:"destination"`
}

func (h *Handlers) PostSubscribe(c echo.Context) error {
	page, err := h.loadPage(c, false)
	if err != nil {
		return err
	}

	var req PostSubscribeRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}

	channel := entities.StatusPageSubscriberChannel(strings.ToUpper(strings.TrimSpace(req.Channel)))
	if channel == "" {
		channel = entities.StatusPageSubscriberChannelEmail
	}

	if err := validateSubscription(channel, req.Email, req.Destination); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.StatusPageService.Subscribe(c.Request().Context(), page, channel, req.Email, req.Destination); err != nil {
		h.Logger.WithError(err).Error("failed to subscribe to status page")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusAccepted)
}

func validateSubscription(channel entities.StatusPageSubscriberChannel, email, destination string) error {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 254 {
		return errors.New("a valid email is required")
	}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return errors.New("a valid email is required")
	}

	switch channel {
	case entities.StatusPageSubscriberChannelEmail:
		return nil
	case entities.StatusPageSubscriberChannelWebhook, entities.StatusPageSubscriberChannelSlack:
		u, err := url.Parse(strings.TrimSpace(destination))
		if err != nil || u.Scheme != "https" || u.Host == "" || len(destination) > 2048 {
			return errors.New("destination must be a valid https URL")
		}
		if channel == entities.StatusPageSubscriberChannelSlack && u.Hostname() != "hooks.slack.com" {
			return errors.New("destination must be a Slack incoming webhook URL (https://hooks.slack.com/...)")
		}

		return nil
	default:
		return fmt.Errorf("unsupported channel %q", channel)
	}
}

func (h *Handlers) GetVerifySubscriber(c echo.Context) error {
	page, err := h.loadPage(c, false)
	if err != nil {
		return err
	}

	token := c.Param("token")
	if token == "" || len(token) > 128 {
		return echo.ErrNotFound
	}

	if err := h.StatusPageService.VerifySubscriber(c.Request().Context(), page, token); err != nil {
		if errors.Is(err, statuspage.ErrInvalidToken) {
			return echo.ErrNotFound
		}

		h.Logger.WithError(err).Error("failed to verify subscriber")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

package statuspages

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/opsway-io/backend/internal/maintenance"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type PublicHandlers struct {
	StatusPageBaseURL  string
	StatusPageService  statuspage.Service
	IncidentService    incident.Service
	MaintenanceService maintenance.Service
	EmailSender        email.Sender
}

func RegisterPublic(
	e *echo.Group,
	logger *logrus.Entry,
	statusPageBaseURL string,
	statusPageService statuspage.Service,
	incidentService incident.Service,
	maintenanceService maintenance.Service,
	emailSender email.Sender,
) {
	h := &PublicHandlers{
		StatusPageBaseURL:  statusPageBaseURL,
		StatusPageService:  statusPageService,
		IncidentService:    incidentService,
		MaintenanceService: maintenanceService,
		EmailSender:        emailSender,
	}

	publicGroup := e.Group("/public/status-pages")
	publicGroup.GET("/:domain", h.GetPublicStatusPage)
	publicGroup.POST("/:domain/subscribe", h.Subscribe)
	publicGroup.GET("/:domain/subscribe/verify/:token", h.VerifySubscriber)
	publicGroup.DELETE("/:domain/subscribe/:token", h.Unsubscribe)
}

type GetPublicStatusPageRequest struct {
	Domain string `param:"domain" validate:"required"`
}

type PublicMonitor struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // "OPERATIONAL" or "OUTAGE"
	CreatedAt time.Time `json:"createdAt"`
}

type PublicIncident struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	MonitorID   *uint  `json:"monitorId"`
}

type PublicMaintenance struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"startAt"`
	EndAt       time.Time `json:"endAt"`
}

type GetPublicStatusPageResponse struct {
	Name                 string              `json:"name"`
	LogoURL              string              `json:"logoUrl"`
	LogoLink             string              `json:"logoLink"`
	FaviconURL           string              `json:"faviconUrl"`
	Layout               string              `json:"layout"`
	CustomCSS            string              `json:"customCss"`
	HeaderHTML           string              `json:"headerHtml"`
	FooterHTML           string              `json:"footerHtml"`
	CustomComponentsHTML string              `json:"customComponentsHtml"`
	ShowBranding         bool                `json:"showBranding"`
	IsPrivate            bool                `json:"isPrivate"`
	Monitors             []PublicMonitor     `json:"monitors"`
	ActiveIncidents      []PublicIncident    `json:"activeIncidents"`
	ActiveMaintenance    []PublicMaintenance `json:"activeMaintenance"`
	MaintenanceEvents    []PublicMaintenance `json:"maintenanceEvents"`
}

func (h *PublicHandlers) GetPublicStatusPage(c echo.Context) error {
	req, err := helpers.Bind[GetPublicStatusPageRequest](c)
	if err != nil {
		logrus.WithError(err).Debug("failed to bind GetPublicStatusPageRequest")
		return echo.ErrBadRequest
	}

	sp, err := h.StatusPageService.GetByDomain(c.Request().Context(), req.Domain)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		logrus.WithError(err).Error("failed to get public status page")
		return echo.ErrInternalServerError
	}

	// Password validation if status page is private
	if sp.IsPrivate {
		passwordHeader := c.Request().Header.Get("X-Status-Page-Password")
		if passwordHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"message":   "Password required",
				"isPrivate": true,
			})
		}
		err := bcrypt.CompareHashAndPassword([]byte(sp.PasswordHash), []byte(passwordHeader))
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"message":   "Invalid password",
				"isPrivate": true,
			})
		}
	}

	monitors := make([]PublicMonitor, len(sp.Monitors))
	var monitorIDs []uint
	for i, m := range sp.Monitors {
		monitorIDs = append(monitorIDs, m.ID)
		status := "OPERATIONAL"
		if m.State == 0 { // Inactive
			status = "OUTAGE"
		}
		monitors[i] = PublicMonitor{
			ID:        m.ID,
			Name:      m.Name,
			Status:    status,
			CreatedAt: m.CreatedAt,
		}
	}

	activeIncidents := []PublicIncident{}
	incidents, err := h.IncidentService.GetActiveByMonitorIDs(c.Request().Context(), monitorIDs)
	if err == nil {
		for _, inc := range incidents {
			desc := ""
			if inc.Description != nil {
				desc = *inc.Description
			}
			activeIncidents = append(activeIncidents, PublicIncident{
				ID:          inc.ID,
				Title:       inc.Title,
				Description: desc,
				MonitorID:   inc.MonitorID,
			})
		}
	}

	activeMaintenance := []PublicMaintenance{}
	maintenances, err := h.MaintenanceService.GetActiveByMonitorIDs(c.Request().Context(), time.Now(), monitorIDs)
	if err == nil {
		for _, mnt := range maintenances {
			desc := ""
			if mnt.Description != nil {
				desc = *mnt.Description
			}
			activeMaintenance = append(activeMaintenance, PublicMaintenance{
				ID:          mnt.ID,
				Title:       mnt.Title,
				Description: desc,
				StartAt:     mnt.Settings.StartAt,
				EndAt:       mnt.Settings.EndAt,
			})
		}
	}

	maintenanceEvents := []PublicMaintenance{}
	allMaintenances, err := h.MaintenanceService.GetAllByMonitorIDs(c.Request().Context(), monitorIDs)
	if err == nil {
		for _, mnt := range allMaintenances {
			desc := ""
			if mnt.Description != nil {
				desc = *mnt.Description
			}
			maintenanceEvents = append(maintenanceEvents, PublicMaintenance{
				ID:          mnt.ID,
				Title:       mnt.Title,
				Description: desc,
				StartAt:     mnt.Settings.StartAt,
				EndAt:       mnt.Settings.EndAt,
			})
		}
	}

	return c.JSON(http.StatusOK, GetPublicStatusPageResponse{
		Name:                 sp.Name,
		LogoURL:              sp.LogoURL,
		LogoLink:             sp.LogoLink,
		FaviconURL:           sp.FaviconURL,
		Layout:               sp.Layout,
		CustomCSS:            sp.CustomCSS,
		HeaderHTML:           sp.HeaderHTML,
		FooterHTML:           sp.FooterHTML,
		CustomComponentsHTML: sp.CustomComponentsHTML,
		ShowBranding:         sp.ShowBranding,
		IsPrivate:            sp.IsPrivate,
		Monitors:             monitors,
		ActiveIncidents:      activeIncidents,
		ActiveMaintenance:    activeMaintenance,
		MaintenanceEvents:    maintenanceEvents,
	})
}

type SubscribeRequest struct {
	Domain string `param:"domain" validate:"required"`
	Email  string `json:"email" validate:"required,email"`
}

func (h *PublicHandlers) Subscribe(c echo.Context) error {
	req, err := helpers.Bind[SubscribeRequest](c)
	if err != nil {
		logrus.WithError(err).Debug("failed to bind SubscribeRequest")
		return echo.ErrBadRequest
	}

	sp, err := h.StatusPageService.GetByDomain(c.Request().Context(), req.Domain)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		return echo.ErrInternalServerError
	}

	token := uuid.New().String()
	err = h.StatusPageService.Subscribe(c.Request().Context(), sp.ID, req.Email, token)
	if err != nil {
		logrus.WithError(err).Error("failed to create subscriber")
		return echo.ErrInternalServerError
	}

	// Dispatch verification email
	verificationURL := fmt.Sprintf("%s/%s/subscribe/verify/%s", h.StatusPageBaseURL, req.Domain, token)
	err = h.EmailSender.Send(c.Request().Context(), "", req.Email, &templates.StatuspageVerifyTemplate{
		StatusPageName:  sp.Name,
		VerificationURL: verificationURL,
	})
	if err != nil {
		logrus.WithError(err).Error("failed to dispatch verification email")
	}

	logrus.WithFields(logrus.Fields{
		"email": req.Email,
		"token": token,
	}).Info("Dispatched verification email")

	return c.NoContent(http.StatusOK)
}

type VerifySubscriberRequest struct {
	Domain string `param:"domain" validate:"required"`
	Token  string `param:"token" validate:"required"`
}

func (h *PublicHandlers) VerifySubscriber(c echo.Context) error {
	req, err := helpers.Bind[VerifySubscriberRequest](c)
	if err != nil {
		logrus.WithError(err).Debug("failed to bind VerifySubscriberRequest")
		return echo.ErrBadRequest
	}

	sp, err := h.StatusPageService.GetByDomain(c.Request().Context(), req.Domain)
	if err != nil {
		if errors.Is(err, statuspage.ErrNotFound) {
			return echo.ErrNotFound
		}
		return echo.ErrInternalServerError
	}

	sub, err := h.StatusPageService.GetSubscriberByToken(c.Request().Context(), req.Token)
	if err != nil {
		logrus.WithError(err).Error("failed to get subscriber")
		return echo.ErrInternalServerError
	}

	err = h.StatusPageService.VerifySubscriber(c.Request().Context(), req.Token)
	if err != nil {
		logrus.WithError(err).Error("failed to verify subscriber")
		return echo.ErrInternalServerError
	}

	// Dispatch subscription confirmed email
	statusPageURL := fmt.Sprintf("%s/%s", h.StatusPageBaseURL, sp.Domain)
	unsubscribeURL := fmt.Sprintf("%s/%s/subscribe/%s", h.StatusPageBaseURL, sp.Domain, sub.Token)
	err = h.EmailSender.Send(c.Request().Context(), "", sub.Email, &templates.SubscriptionConfirmedTemplate{
		StatusPageName: sp.Name,
		StatusPageURL:  statusPageURL,
		UnsubscribeURL: unsubscribeURL,
	})
	if err != nil {
		logrus.WithError(err).Error("failed to dispatch subscription confirmed email")
	}

	return c.JSON(http.StatusOK, map[string]bool{"verified": true})
}

func (h *PublicHandlers) Unsubscribe(c echo.Context) error {
	req, err := helpers.Bind[VerifySubscriberRequest](c) // reusing VerifySubscriberRequest struct since it has Domain and Token
	if err != nil {
		logrus.WithError(err).Debug("failed to bind unsubscribe request")
		return echo.ErrBadRequest
	}

	err = h.StatusPageService.Unsubscribe(c.Request().Context(), req.Token)
	if err != nil {
		logrus.WithError(err).Error("failed to unsubscribe")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, map[string]bool{"unsubscribed": true})
}

package monitors

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PreviewOpenAPIRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gte=0"`
	URL    string `json:"url" validate:"required,url"`
}

type PreviewOpenAPIResponse struct {
	Endpoints []PreviewOpenAPIEndpoint `json:"endpoints"`
}

type PreviewOpenAPIEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Summary     string `json:"summary"`
	RequestBody string `json:"requestBody,omitempty"`
	StatusCode  string `json:"statusCode"`
}

func (h *Handlers) PreviewOpenAPI(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PreviewOpenAPIRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PreviewOpenAPIRequest")
		return echo.ErrBadRequest
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid URL")
	}

	doc, err := loader.LoadFromURI(parsedURL)
	if err != nil {
		c.Log.WithError(err).Error("failed to load openapi spec")
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to fetch or parse OpenAPI spec: %v", err))
	}

	endpoints := make([]PreviewOpenAPIEndpoint, 0)
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			endpoint := PreviewOpenAPIEndpoint{
				Method:  method,
				Path:    path,
				Summary: operation.Summary,
			}

			// Find expected status code (first 2xx or just 200)
			statusCode := "200"
			for code := range operation.Responses.Map() {
				if len(code) > 0 && code[0] == '2' {
					statusCode = code
					break
				}
			}
			endpoint.StatusCode = statusCode

			// Get example request body if available
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				content := operation.RequestBody.Value.Content
				if jsonContent, ok := content["application/json"]; ok {
					if jsonContent.Example != nil {
						endpoint.RequestBody = fmt.Sprintf("%v", jsonContent.Example)
					}
				}
			}

			endpoints = append(endpoints, endpoint)
		}
	}

	return c.JSON(http.StatusOK, PreviewOpenAPIResponse{
		Endpoints: endpoints,
	})
}

type PostMonitorsBulkRequest struct {
	TeamID   uint                 `param:"teamId" validate:"required,numeric,gte=0"`
	Monitors []PostMonitorRequest `json:"monitors" validate:"required,dive"`
}

func (h *Handlers) PostMonitorsBulk(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostMonitorsBulkRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostMonitorsBulkRequest")
		return echo.ErrBadRequest
	}

	ctx := c.Request().Context()
	teamEntity, err := h.TeamService.GetByID(ctx, req.TeamID)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get team")
		return echo.ErrInternalServerError
	}

	limit := teamEntity.GetMonitorLimit()
	if limit != -1 {
		count, err := h.MonitorService.CountByTeamID(ctx, req.TeamID)
		if err != nil {
			c.Log.WithError(err).Debug("failed to get monitor count")
			return echo.ErrInternalServerError
		}
		if count+int64(len(req.Monitors)) > int64(limit) {
			return echo.NewHTTPError(http.StatusPaymentRequired, "Payment Required: Adding these monitors exceeds your plan limit")
		}
	}

	entitiesToCreate := make([]*entities.Monitor, 0, len(req.Monitors))

	for _, mReq := range req.Monitors {
		headers := make([]entities.MonitorSettingsHeader, len(mReq.Settings.Headers))
		for j, h := range mReq.Settings.Headers {
			headers[j] = entities.MonitorSettingsHeader{
				Key:   h.Key,
				Value: h.Value,
			}
		}

		assertions := make([]entities.MonitorAssertion, len(mReq.Assertions))
		for j, a := range mReq.Assertions {
			assertions[j] = entities.MonitorAssertion{
				Source:   a.Source,
				Operator: a.Operator,
				Target:   a.Target,
				Property: a.Property,
			}
		}

		m := &entities.Monitor{
			TeamID: req.TeamID,
			Name:   mReq.Name,
			Settings: entities.MonitorSettings{
				Method:  mReq.Settings.Method,
				URL:     mReq.Settings.URL,
				Headers: headers,
				Body: entities.MonitorSettingsBody{
					Type: mReq.Settings.Body.Type,
				},
				TLS: entities.MonitorSettingsTLS{
					Enabled:                 mReq.Settings.TLS.Enabled,
					VerifyHostname:          mReq.Settings.TLS.VerifyHostname,
					CheckExpiration:         mReq.Settings.TLS.CheckExpiration,
					ExpirationThresholdDays: mReq.Settings.TLS.ExpirationThresholdDays,
				},
				Auth: entities.MonitorSettingsAuth{
					Method:       mReq.Settings.Auth.Method,
					TokenURL:     mReq.Settings.Auth.TokenURL,
					ClientID:     mReq.Settings.Auth.ClientID,
					ClientSecret: mReq.Settings.Auth.ClientSecret,
					Username:     mReq.Settings.Auth.Username,
					Password:     mReq.Settings.Auth.Password,
				},
				Locations: mReq.Settings.Locations,
			},
			Assertions: assertions,
		}

		m.Settings.SetFrequencySeconds(mReq.Settings.FrequencySeconds)
		m.Settings.Body.SetContentString(mReq.Settings.Body.Content)

		entitiesToCreate = append(entitiesToCreate, m)
	}

	if err := h.MonitorService.CreateBulk(c.Request().Context(), entitiesToCreate); err != nil {
		c.Log.WithError(err).Error("failed to bulk create monitors")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}

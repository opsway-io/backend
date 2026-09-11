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
	Auth      *PreviewOpenAPIAuth      `json:"auth,omitempty"`
}

type PreviewOpenAPIAuth struct {
	Method   string `json:"method"`
	TokenURL string `json:"tokenUrl,omitempty"`
}

type PreviewOpenAPIAssertion struct {
	Source   string `json:"source"`
	Property string `json:"property,omitempty"`
	Operator string `json:"operator"`
	Target   string `json:"target,omitempty"`
}

type PreviewOpenAPIEndpoint struct {
	Method      string                    `json:"method"`
	Path        string                    `json:"path"`
	Summary     string                    `json:"summary"`
	RequestBody string                    `json:"requestBody,omitempty"`
	StatusCode  string                    `json:"statusCode"`
	Assertions  []PreviewOpenAPIAssertion `json:"assertions"`
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

	var auth *PreviewOpenAPIAuth
	if doc.Components != nil && doc.Components.SecuritySchemes != nil {
		for _, schemeRef := range doc.Components.SecuritySchemes {
			if schemeRef.Value != nil {
				if schemeRef.Value.Type == "oauth2" && schemeRef.Value.Flows != nil && schemeRef.Value.Flows.ClientCredentials != nil {
					auth = &PreviewOpenAPIAuth{
						Method:   "OAUTH2_CLIENT_CREDENTIALS",
						TokenURL: schemeRef.Value.Flows.ClientCredentials.TokenURL,
					}
					break
				} else if schemeRef.Value.Type == "http" && schemeRef.Value.Scheme == "basic" {
					auth = &PreviewOpenAPIAuth{
						Method: "BASIC",
					}
					break
				}
			}
		}
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

			assertions := []PreviewOpenAPIAssertion{}
			assertions = append(assertions, PreviewOpenAPIAssertion{
				Source:   "STATUS_CODE",
				Operator: "EQUAL",
				Target:   statusCode,
			})

			responseRef := operation.Responses.Map()[statusCode]
			if responseRef != nil && responseRef.Value != nil {
				if content, ok := responseRef.Value.Content["application/json"]; ok {
					if content.Schema != nil && content.Schema.Value != nil {
						schema := content.Schema.Value
						if schema.Properties != nil {
							for propName, propRef := range schema.Properties {
								assertions = append(assertions, PreviewOpenAPIAssertion{
									Source:   "JSON_BODY",
									Property: "$." + propName,
									Operator: "HAS_KEY",
								})

								if propRef.Value != nil && propRef.Value.Properties != nil {
									for subPropName := range propRef.Value.Properties {
										assertions = append(assertions, PreviewOpenAPIAssertion{
											Source:   "JSON_BODY",
											Property: "$." + propName + "." + subPropName,
											Operator: "HAS_KEY",
										})
									}
								}
							}
						}
					}
				}
			}

			endpoint.Assertions = assertions
			endpoints = append(endpoints, endpoint)
		}
	}

	return c.JSON(http.StatusOK, PreviewOpenAPIResponse{
		Endpoints: endpoints,
		Auth:      auth,
	})
}

type PostMonitorsBulkMonitorRequest struct {
	Name       string             `json:"name" validate:"required,max=255"`
	Settings   MonitorSettings    `json:"settings" validate:"required,dive"`
	Steps      []MonitorStep      `json:"steps" validate:"required,dive"`
}

type PostMonitorsBulkRequest struct {
	TeamID   uint                             `param:"teamId" validate:"required,numeric,gte=0"`
	Monitors []PostMonitorsBulkMonitorRequest `json:"monitors" validate:"required,dive"`
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
		steps := make([]entities.MonitorStep, len(mReq.Steps))
		for i, s := range mReq.Steps {
			headers := make([]entities.MonitorStepHeader, len(s.Headers))
			for j, h := range s.Headers {
				headers[j] = entities.MonitorStepHeader{
					Key:   h.Key,
					Value: h.Value,
				}
			}

			assertions := make([]entities.MonitorAssertion, len(s.Assertions))
			for j, a := range s.Assertions {
				assertions[j] = entities.MonitorAssertion{
					Source:   a.Source,
					Operator: a.Operator,
					Target:   a.Target,
					Property: a.Property,
				}
			}

			variables := make([]entities.MonitorVariable, len(s.Variables))
			for j, v := range s.Variables {
				variables[j] = entities.MonitorVariable{
					Name:     v.Name,
					Source:   v.Source,
					Property: v.Property,
				}
			}
			
			body := entities.MonitorStepBody{
				Type: s.Body.Type,
			}
			body.SetContentString(s.Body.Content)
			
			steps[i] = entities.MonitorStep{
				Name:       s.Name,
				OrderIndex: i,
				Method:     s.Method,
				URL:        s.URL,
				Headers:    headers,
				Body:       body,
				Assertions: assertions,
				Variables:  variables,
			}
		}

		m := &entities.Monitor{
			TeamID: req.TeamID,
			Name:   mReq.Name,
			Settings: entities.MonitorSettings{
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
			Steps: steps,
		}

		m.Settings.SetFrequencySeconds(mReq.Settings.FrequencySeconds)

		entitiesToCreate = append(entitiesToCreate, m)
	}

	if err := h.MonitorService.CreateBulk(c.Request().Context(), entitiesToCreate); err != nil {
		c.Log.WithError(err).Error("failed to bulk create monitors")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusCreated)
}

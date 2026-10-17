package monitors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

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
	if doc.Paths != nil {
		paths := make([]string, 0, len(doc.Paths.Map()))
		for p := range doc.Paths.Map() {
			paths = append(paths, p)
		}
		sort.Strings(paths)

		for _, path := range paths {
			pathItem := doc.Paths.Find(path)
			if pathItem == nil {
				continue
			}

			operations := pathItem.Operations()
			methods := make([]string, 0, len(operations))
			for m := range operations {
				methods = append(methods, m)
			}
			sort.Strings(methods)

			for _, method := range methods {
				operation := operations[method]
				if operation == nil {
					continue
				}

				endpoint := PreviewOpenAPIEndpoint{
					Method:  method,
					Path:    path,
					Summary: operation.Summary,
				}

				// Find expected status code (prefer 200, then lowest 2xx, fallback 200)
				statusCode := "200"
				if operation.Responses != nil {
					var codes2xx []string
					for code := range operation.Responses.Map() {
						if len(code) == 3 && code[0] == '2' {
							codes2xx = append(codes2xx, code)
						}
					}
					sort.Strings(codes2xx)
					if len(codes2xx) > 0 {
						has200 := false
						for _, c := range codes2xx {
							if c == "200" {
								has200 = true
								break
							}
						}
						if has200 {
							statusCode = "200"
						} else {
							statusCode = codes2xx[0]
						}
					}
				}
				endpoint.StatusCode = statusCode

				// Get example request body if available
				if operation.RequestBody != nil && operation.RequestBody.Value != nil {
					content := operation.RequestBody.Value.Content
					if jsonContent, ok := content["application/json"]; ok {
						if jsonContent.Example != nil {
							if str, ok := jsonContent.Example.(string); ok {
								endpoint.RequestBody = str
							} else if b, err := json.Marshal(jsonContent.Example); err == nil {
								endpoint.RequestBody = string(b)
							} else {
								endpoint.RequestBody = fmt.Sprintf("%v", jsonContent.Example)
							}
						}
					}
				}

				assertions := []PreviewOpenAPIAssertion{
					{
						Source:   "STATUS_CODE",
						Operator: "EQUAL",
						Target:   statusCode,
					},
				}

				if operation.Responses != nil {
					responseRef := operation.Responses.Map()[statusCode]
					if responseRef != nil && responseRef.Value != nil {
						var responseSchema *openapi3.Schema
						if content, ok := responseRef.Value.Content["application/json"]; ok && content.Schema != nil && content.Schema.Value != nil {
							responseSchema = content.Schema.Value
						} else {
							for mediaType, mediaTypeObj := range responseRef.Value.Content {
								if isJSONMediaType(mediaType) && mediaTypeObj.Schema != nil && mediaTypeObj.Schema.Value != nil {
									responseSchema = mediaTypeObj.Schema.Value
									break
								}
							}
						}

						if responseSchema != nil {
							schemaAssertions := extractSchemaAssertions(responseSchema, "$", 0, 8, make(map[*openapi3.Schema]bool))
							assertions = append(assertions, deduplicateAssertions(schemaAssertions)...)
						}
					}
				}

				endpoint.Assertions = assertions
				endpoints = append(endpoints, endpoint)
			}
		}
	}

	return c.JSON(http.StatusOK, PreviewOpenAPIResponse{
		Endpoints: endpoints,
		Auth:      auth,
	})
}

func isJSONMediaType(mt string) bool {
	mt = strings.ToLower(strings.TrimSpace(mt))
	return strings.Contains(mt, "application/json") || strings.HasSuffix(mt, "+json")
}

func extractSchemaAssertions(
	schema *openapi3.Schema,
	prefix string,
	depth int,
	maxDepth int,
	activeStack map[*openapi3.Schema]bool,
) []PreviewOpenAPIAssertion {
	if schema == nil || depth > maxDepth || activeStack[schema] {
		return nil
	}

	activeStack[schema] = true
	defer func() {
		delete(activeStack, schema)
	}()

	var assertions []PreviewOpenAPIAssertion

	// Handle allOf composition
	for _, subRef := range schema.AllOf {
		if subRef != nil && subRef.Value != nil {
			assertions = append(assertions, extractSchemaAssertions(subRef.Value, prefix, depth, maxDepth, activeStack)...)
		}
	}

	// Handle oneOf / anyOf if no explicit properties
	if schema.Properties == nil && len(schema.AllOf) == 0 {
		if len(schema.OneOf) > 0 && schema.OneOf[0] != nil && schema.OneOf[0].Value != nil {
			assertions = append(assertions, extractSchemaAssertions(schema.OneOf[0].Value, prefix, depth, maxDepth, activeStack)...)
		} else if len(schema.AnyOf) > 0 && schema.AnyOf[0] != nil && schema.AnyOf[0].Value != nil {
			assertions = append(assertions, extractSchemaAssertions(schema.AnyOf[0].Value, prefix, depth, maxDepth, activeStack)...)
		}
	}

	// Handle array types
	isArray := (schema.Type != nil && schema.Type.Permits("array")) || schema.Items != nil
	if isArray {
		if prefix == "$" {
			assertions = append(assertions, PreviewOpenAPIAssertion{
				Source:   "JSON_BODY",
				Property: "$",
				Operator: "NOT_NULL",
			})
			if schema.Items != nil && schema.Items.Value != nil {
				assertions = append(assertions, extractSchemaAssertions(schema.Items.Value, "$[0]", depth+1, maxDepth, activeStack)...)
			}
		} else if schema.Items != nil && schema.Items.Value != nil {
			assertions = append(assertions, extractSchemaAssertions(schema.Items.Value, prefix+"[0]", depth+1, maxDepth, activeStack)...)
		}
	}

	// Handle object properties
	if schema.Properties != nil {
		requiredMap := make(map[string]bool, len(schema.Required))
		for _, req := range schema.Required {
			requiredMap[req] = true
		}

		propNames := make([]string, 0, len(schema.Properties))
		for propName := range schema.Properties {
			propNames = append(propNames, propName)
		}
		sort.Strings(propNames)

		for _, propName := range propNames {
			propRef := schema.Properties[propName]
			var propPath string
			if prefix == "$" {
				propPath = "$." + propName
			} else {
				propPath = prefix + "." + propName
			}

			operator := "HAS_KEY"
			if requiredMap[propName] && (propRef.Value == nil || !propRef.Value.PermitsNull()) {
				operator = "NOT_NULL"
			}

			assertions = append(assertions, PreviewOpenAPIAssertion{
				Source:   "JSON_BODY",
				Property: propPath,
				Operator: operator,
			})

			if propRef.Value != nil {
				subAssertions := extractSchemaAssertions(propRef.Value, propPath, depth+1, maxDepth, activeStack)
				assertions = append(assertions, subAssertions...)
			}
		}
	}

	return deduplicateAssertions(assertions)
}

func deduplicateAssertions(assertions []PreviewOpenAPIAssertion) []PreviewOpenAPIAssertion {
	seen := make(map[string]bool, len(assertions))
	result := make([]PreviewOpenAPIAssertion, 0, len(assertions))
	for _, a := range assertions {
		key := a.Source + "|" + a.Property + "|" + a.Operator + "|" + a.Target
		if !seen[key] {
			seen[key] = true
			result = append(result, a)
		}
	}
	return result
}

type PostMonitorsBulkMonitorRequest struct {
	Name     string          `json:"name" validate:"required,max=255"`
	Settings MonitorSettings `json:"settings" validate:"required,dive"`
	Steps    []MonitorStep   `json:"steps" validate:"required,dive"`
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
			State:  entities.MonitorStateActive,
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

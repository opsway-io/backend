package monitors

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/monitor/openapi"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type PostOpenAPIPreviewRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gte=0"`
	URL    string `json:"url" validate:"required,max=2048"`
}

func (h *Handlers) PostOpenAPIPreview(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostOpenAPIPreviewRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostOpenAPIPreviewRequest")

		return echo.ErrBadRequest
	}

	preview, err := openapi.Fetch(c.Request().Context(), nil, req.URL)
	if err != nil {
		c.Log.WithError(err).Debug("failed to load OpenAPI document")

		if errors.Is(err, openapi.ErrInvalidURL) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid url")
		}

		if errors.Is(err, openapi.ErrInvalidSpec) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "the document is not a valid OpenAPI 3 or Swagger 2 specification")
		}

		return echo.NewHTTPError(http.StatusBadGateway, "failed to fetch the OpenAPI document")
	}

	return c.JSON(http.StatusOK, preview)
}

type GetProberLocationsResponse struct {
	Locations []string `json:"locations"`
}

func (h *Handlers) GetProberLocations(c hs.AuthenticatedContext) error {
	return c.JSON(http.StatusOK, GetProberLocationsResponse{
		Locations: h.MonitorService.Locations(),
	})
}

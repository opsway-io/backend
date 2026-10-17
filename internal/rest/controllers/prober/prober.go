package prober

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/prober"
	"github.com/sirupsen/logrus"
)

func Register(e *echo.Group, logger *logrus.Entry, proberService prober.Service) {
	h := &Handlers{
		proberService: proberService,
		logger:        logger,
	}

	e.GET("/prober/locations", h.GetLocations)
}

type Handlers struct {
	proberService prober.Service
	logger        *logrus.Entry
}

type LocationDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	Status      string `json:"status"`
}

type GetLocationsResponse struct {
	Locations []string         `json:"locations"`
	Details   []LocationDetail `json:"details"`
}

func (h *Handlers) GetLocations(c echo.Context) error {
	ctx := c.Request().Context()
	activeLocs, err := h.proberService.GetActiveLocations(ctx)
	if err != nil && h.logger != nil {
		h.logger.WithError(err).Warn("Failed to fetch active prober locations")
	}

	locations := make([]string, len(activeLocs))
	details := make([]LocationDetail, len(activeLocs))

	for i, loc := range activeLocs {
		locations[i] = loc.ID
		details[i] = LocationDetail{
			ID:          loc.ID,
			Name:        loc.Name,
			CountryCode: loc.CountryCode,
			Status:      loc.Status,
		}
	}

	return c.JSON(http.StatusOK, GetLocationsResponse{
		Locations: locations,
		Details:   details,
	})
}

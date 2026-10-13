package prober

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func Register(e *echo.Group, logger *logrus.Entry, availableLocations []string) {
	h := &Handlers{
		AvailableLocations: availableLocations,
	}

	e.GET("/prober/locations", h.GetLocations)
}

type Handlers struct {
	AvailableLocations []string
}

type GetLocationsResponse struct {
	Locations []string `json:"locations"`
}

func (h *Handlers) GetLocations(c echo.Context) error {
	return c.JSON(http.StatusOK, GetLocationsResponse{
		Locations: h.AvailableLocations,
	})
}

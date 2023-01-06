package healthz

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func Register(e *echo.Group, logger *logrus.Entry,
) {
	e.GET("/healthz", func(c echo.Context) error {
		logger.Debug("healthz check")

		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})
}

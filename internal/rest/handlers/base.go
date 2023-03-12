package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type BaseHandlerFunc func(c BaseContext) error

type BaseContext struct {
	echo.Context
	Log *logrus.Entry
}

func BaseHandlerFactory(logger *logrus.Entry) func(handler BaseHandlerFunc) func(c echo.Context) error {
	return func(handler BaseHandlerFunc) func(c echo.Context) error {
		return func(c echo.Context) error {
			return handler(BaseContext{
				Context: c,
				Log:     logger,
			})
		}
	}
}

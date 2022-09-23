package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type BaseHandlerFunc func(ctx BaseContext) error

type BaseContext struct {
	echo.Context
	Log *logrus.Entry
}

func BaseHandler(handler BaseHandlerFunc, logger *logrus.Entry) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return handler(BaseContext{
			Context: ctx,
			Log:     logger,
		})
	}
}

package wrappers

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type StandardHandlerFunc func(ctx echo.Context, logger *zap.Logger) error

func StandardHandler(handler StandardHandlerFunc, logger *zap.Logger) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return handler(ctx, logger)
	}
}

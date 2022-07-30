package helpers

import (
	"github.com/creasty/defaults"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func Bind[V interface{}](ctx echo.Context) (*V, error) {
	var data V
	if err := ctx.Bind(&data); err != nil {
		return nil, errors.Wrap(err, "failed to bind request")
	}

	if err := ctx.Validate(&data); err != nil {
		return nil, errors.Wrap(err, "request failed validation")
	}

	if err := defaults.Set(&data); err != nil {
		return nil, errors.Wrap(err, "failed to set defaults")
	}

	return &data, nil
}

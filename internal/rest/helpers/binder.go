package helpers

import (
	"github.com/creasty/defaults"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func Bind[V interface{}](c echo.Context) (*V, error) {
	var data V
	if err := c.Bind(&data); err != nil {
		return nil, errors.Wrap(err, "failed to bind request")
	}

	if err := defaults.Set(&data); err != nil {
		return nil, errors.Wrap(err, "failed to set defaults")
	}

	if err := c.Validate(&data); err != nil {
		return nil, errors.Wrap(err, "request failed validation")
	}

	return &data, nil
}

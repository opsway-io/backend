package authentication

import (
	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/jwt"
	w "github.com/opsway-io/backend/internal/rest/wrappers"
	"github.com/opsway-io/backend/internal/user"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	UserService user.Service
	JWTService  jwt.Service
}

func Register(e *echo.Group, logger *logrus.Entry, userService user.Service, jwtService jwt.Service) {
	h := &Handlers{
		UserService: userService,
		JWTService:  jwtService,
	}

	g := e.Group("/authentication")

	g.POST("/login", w.StandardHandler(h.PostLogin, logger))
	g.POST("/refresh", w.StandardHandler(h.PostRefresh, logger))
}

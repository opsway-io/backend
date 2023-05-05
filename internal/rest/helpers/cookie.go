package helpers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/authentication"
)

const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"
)

type CookieService interface {
	SetAccessToken(c echo.Context, value string) error
	SetRefreshToken(c echo.Context, value string) error
	GetAccessToken(c echo.Context) (*http.Cookie, error)
	GetRefreshToken(c echo.Context) (*http.Cookie, error)
}

type CookieServiceImpl struct {
	authConfig *authentication.Config
}

func NewCookieService(authConfig *authentication.Config) CookieService {
	return &CookieServiceImpl{
		authConfig: authConfig,
	}
}

func (s *CookieServiceImpl) SetAccessToken(c echo.Context, value string) error {
	cookie := new(http.Cookie)
	cookie.Name = AccessTokenCookieName
	cookie.Value = value
	cookie.Expires = time.Now().Add(s.authConfig.RefreshExpiresIn)
	cookie.Domain = s.authConfig.CookieDomain
	cookie.Path = "/"

	c.SetCookie(cookie)

	return nil
}

func (s *CookieServiceImpl) SetRefreshToken(c echo.Context, value string) error {
	cookie := new(http.Cookie)
	cookie.Name = RefreshTokenCookieName
	cookie.Value = value
	cookie.Expires = time.Now().Add(s.authConfig.RefreshExpiresIn)
	cookie.MaxAge = int(s.authConfig.RefreshExpiresIn.Seconds())
	cookie.Domain = s.authConfig.CookieDomain
	cookie.Path = "/"

	c.SetCookie(cookie)

	return nil
}

func (s *CookieServiceImpl) GetAccessToken(c echo.Context) (*http.Cookie, error) {
	return c.Cookie(AccessTokenCookieName)
}

func (s *CookieServiceImpl) GetRefreshToken(c echo.Context) (*http.Cookie, error) {
	return c.Cookie(RefreshTokenCookieName)
}

package authentication

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
	"github.com/opsway-io/backend/internal/team"
	"github.com/opsway-io/backend/internal/user"
	"k8s.io/utils/pointer"
)

type PostLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=4,max=255"`
}

type PostRegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=4,max=255"`
}

type PostLoginResponse struct {
	User         PostLoginResponseUser `json:"user"`
}

type PostLoginResponseUser struct {
	ID          uint                    `json:"id"`
	Name        string                  `json:"name"`
	DisplayName *string                 `json:"displayName"`
	Email       string                  `json:"email"`
	AvatarURL   *string                 `json:"avatarUrl"`
	Teams       []PostLoginResponseTeam `json:"teams"`
	CreatedAt   time.Time               `json:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

type PostLoginResponseTeam struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
}

func (h *Handlers) PostLogin(c hs.BaseContext) error {
	req, err := helpers.Bind[PostLoginRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostLoginRequest")

		return echo.ErrBadRequest
	}

	u, err := h.UserService.GetUserAndTeamsByEmailAddress(c.Request().Context(), req.Email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			c.Log.WithError(err).Debug("user not found")
			return echo.ErrUnauthorized
		}
		
		c.Log.WithError(err).Error("failed to get user")
		return echo.ErrInternalServerError
	}

	c.Log = c.Log.WithField("user_id", u.ID)

	if ok := u.CheckPassword(req.Password); !ok {
		c.Log.Debug("password invalid")

		return echo.ErrUnauthorized
	}

	accessToken, refreshToken, err := h.AuthenticationService.Generate(c.Request().Context(), u)
	if err != nil {
		c.Log.WithError(err).Debug("failed to generate access and refresh token for user")

		return echo.ErrInternalServerError
	}

	c.Log.Info("user authenticated")

	h.CookieService.SetAccessToken(c, accessToken)
	h.CookieService.SetRefreshToken(c, refreshToken)

	return c.JSON(http.StatusOK, newPostLoginResponse(
		u,
		h.UserService,
		h.TeamService,
	))
}

func newPostLoginResponse(user *entities.User, userService user.Service, teamService team.Service) PostLoginResponse {
	teams := make([]PostLoginResponseTeam, len(user.Teams))
	for i, team := range user.Teams {
		teams[i] = PostLoginResponseTeam{
			ID:          team.ID,
			Name:        team.Name,
			DisplayName: team.DisplayName,
		}

		if team.HasAvatar {
			teams[i].AvatarURL = pointer.String(teamService.GetAvatarURLByID(team.ID))
		}
	}

	res := PostLoginResponse{
		User: PostLoginResponseUser{
			ID:          user.ID,
			Name:        user.Name,
			DisplayName: user.DisplayName,
			Email:       user.Email,
			Teams:       teams,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		},
	}

	if user.HasAvatar {
		res.User.AvatarURL = pointer.String(userService.GetAvatarURLByID(user.ID))
	}

	return res
}

func (h *Handlers) PostRegister(c hs.BaseContext) error {
	req, err := helpers.Bind[PostRegisterRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostRegisterRequest")

		return echo.ErrBadRequest
	}

	u := &entities.User{
		Name: req.Name,
	}
	u.SetEmail(req.Email)
	if err := u.SetPassword(req.Password); err != nil {
		c.Log.WithError(err).Debug("failed to set password")

		return echo.ErrInternalServerError
	}

	if err := h.UserService.Create(c.Request().Context(), u); err != nil {
		if errors.Is(err, user.ErrEmailAlreadyExists) {
			return echo.NewHTTPError(http.StatusConflict, "user already exists")
		}

		c.Log.WithError(err).Debug("failed to create user")

		return echo.ErrInternalServerError
	}

	c.Log = c.Log.WithField("user_id", u.ID)

	accessToken, refreshToken, err := h.AuthenticationService.Generate(c.Request().Context(), u)
	if err != nil {
		c.Log.WithError(err).Debug("failed to generate access and refresh token for user")

		return echo.ErrInternalServerError
	}

	c.Log.Info("user registered")

	h.CookieService.SetAccessToken(c, accessToken)
	h.CookieService.SetRefreshToken(c, refreshToken)

	return c.JSON(http.StatusOK, newPostLoginResponse(
		u,
		h.UserService,
		h.TeamService,
	))
}

type PostForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

func (h *Handlers) PostForgotPassword(c hs.BaseContext) error {
	req, err := helpers.Bind[PostForgotPasswordRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostForgotPasswordRequest")
		return echo.ErrBadRequest
	}

	user, err := h.UserService.GetUserAndTeamsByEmailAddress(c.Request().Context(), req.Email)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get user")
		// We return 200 OK anyway to prevent user enumeration
		return c.NoContent(http.StatusOK)
	}

	if err := h.UserService.RequestPasswordReset(c.Request().Context(), user.ID); err != nil {
		c.Log.WithError(err).Error("failed to request password reset")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}

type PostResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=4,max=255"`
}

func (h *Handlers) PostResetPassword(c hs.BaseContext) error {
	req, err := helpers.Bind[PostResetPasswordRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostResetPasswordRequest")
		return echo.ErrBadRequest
	}

	if err := h.UserService.ChangePasswordWithResetToken(c.Request().Context(), req.Token, req.Password); err != nil {
		c.Log.WithError(err).Error("failed to change password with reset token")
		return echo.ErrBadRequest
	}

	return c.NoContent(http.StatusOK)
}

func (h *Handlers) PostLogout(c hs.BaseContext) error {
	// Set cookies to expire immediately
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	
	return c.NoContent(http.StatusOK)
}

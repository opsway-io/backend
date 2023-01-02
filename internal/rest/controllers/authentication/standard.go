package authentication

import (
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
	Password string `json:"password" validate:"required"`
}

type PostLoginResponse struct {
	AccessToken  string                `json:"accessToken"`
	RefreshToken string                `json:"refreshToken"`
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

func (h *Handlers) PostLogin(ctx hs.BaseContext) error {
	req, err := helpers.Bind[PostLoginRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PostLoginRequest")

		return echo.ErrBadRequest
	}

	user, err := h.UserService.GetUserAndTeamsByEmailAddress(ctx.Request().Context(), req.Email)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to get user")

		return echo.ErrUnauthorized
	}

	ctx.Log = ctx.Log.WithField("user_id", user.ID)

	if ok := user.CheckPassword(req.Password); !ok {
		ctx.Log.Debug("password invalid")

		return echo.ErrUnauthorized
	}

	accessToken, refreshToken, err := h.AuthenticationService.Generate(ctx.Request().Context(), user)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to generate access and refresh token for user")

		return echo.ErrInternalServerError
	}

	ctx.Log.Info("user authenticated")

	return ctx.JSON(http.StatusOK, newPostLoginResponse(user, accessToken, refreshToken, h.UserService, h.TeamService))
}

func newPostLoginResponse(user *entities.User, accessToken, refreshToken string, userService user.Service, teamService team.Service) PostLoginResponse {
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
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
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

type PostRefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type PostRefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (h *Handlers) PostRefreshToken(ctx hs.BaseContext) error {
	req, err := helpers.Bind[PostRefreshTokenRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to bind PostRefreshTokenRequest")

		return echo.ErrBadRequest
	}

	accessToken, refreshToken, err := h.AuthenticationService.Refresh(ctx.Request().Context(), req.RefreshToken)
	if err != nil {
		ctx.Log.WithError(err).Debug("failed to refresh access and refresh token")

		return echo.ErrUnauthorized
	}

	ctx.Log.Info("access and refresh token refreshed")

	return ctx.JSON(http.StatusOK, PostRefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

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
	User PostLoginResponseUser `json:"user"`
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

	user, err := h.UserService.GetUserAndTeamsByEmailAddress(c.Request().Context(), req.Email)
	if err != nil {
		c.Log.WithError(err).Debug("failed to get user")

		return echo.ErrUnauthorized
	}

	c.Log = c.Log.WithField("user_id", user.ID)

	if ok := user.CheckPassword(req.Password); !ok {
		c.Log.Debug("password invalid")

		return echo.ErrUnauthorized
	}

	accessToken, refreshToken, err := h.AuthenticationService.Generate(c.Request().Context(), user)
	if err != nil {
		c.Log.WithError(err).Debug("failed to generate access and refresh token for user")

		return echo.ErrInternalServerError
	}

	c.Log.Info("user authenticated")

	h.CookieService.SetRefreshToken(c, refreshToken)
	h.CookieService.SetAccessToken(c, accessToken)

	return c.JSON(http.StatusOK, newPostLoginResponse(user, h.UserService, h.TeamService))
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

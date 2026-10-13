package apikeys

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/apikey"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetApiKeysRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
}

type GetApiKeysResponse struct {
	ApiKeys []GetApiKeysResponseApiKey `json:"apiKeys"`
}

type GetApiKeysResponseApiKey struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (h *Handlers) GetApiKeys(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetApiKeysRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	keys, err := h.ApiKeyService.GetByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get api keys")

		return echo.ErrInternalServerError
	}

	resp := GetApiKeysResponse{ApiKeys: make([]GetApiKeysResponseApiKey, len(*keys))}
	for i, k := range *keys {
		resp.ApiKeys[i] = GetApiKeysResponseApiKey{
			ID:         k.ID,
			Name:       k.Name,
			Prefix:     k.Prefix,
			LastUsedAt: k.LastUsedAt,
			CreatedAt:  k.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

type PostApiKeyRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gt=0"`
	Name   string `json:"name" validate:"required,max=255"`
}

type PostApiKeyResponse struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	PlaintextKey string    `json:"plaintextKey"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (h *Handlers) PostApiKey(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostApiKeyRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostApiKeyRequest")

		return echo.ErrBadRequest
	}

	key, plaintext, err := h.ApiKeyService.Create(c.Request().Context(), req.TeamID, c.UserID, req.Name)
	if err != nil {
		c.Log.WithError(err).Error("failed to create api key")

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, PostApiKeyResponse{
		ID:           key.ID,
		Name:         key.Name,
		PlaintextKey: plaintext,
		CreatedAt:    key.CreatedAt,
	})
}

type DeleteApiKeyRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gt=0"`
	KeyID  uint `param:"keyId" validate:"required,numeric,gt=0"`
}

func (h *Handlers) DeleteApiKey(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteApiKeyRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.ApiKeyService.Delete(c.Request().Context(), req.TeamID, req.KeyID); err != nil {
		if errors.Is(err, apikey.ErrNotFound) {
			return echo.ErrNotFound
		}

		c.Log.WithError(err).Error("failed to delete api key")

		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusNoContent)
}

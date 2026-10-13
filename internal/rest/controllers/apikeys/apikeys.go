package apikeys

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

type GetApiKeysRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
}

type GetApiKeysResponse struct {
	ApiKeys []GetApiKeysResponseApiKey `json:"apiKeys"`
}

type GetApiKeysResponseApiKey struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

func (h *Handlers) GetApiKeys(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[GetApiKeysRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind GetApiKeysRequest")
		return echo.ErrBadRequest
	}

	keys, err := h.ApiKeyService.GetByTeamID(c.Request().Context(), req.TeamID)
	if err != nil {
		c.Log.WithError(err).Error("failed to get api keys")
		return echo.ErrInternalServerError
	}

	resp := h.newGetApiKeysResponse(keys)

	return c.JSON(http.StatusOK, resp)
}

func (h *Handlers) newGetApiKeysResponse(keys *[]entities.APIKey) *GetApiKeysResponse {
	resp := &GetApiKeysResponse{
		ApiKeys: make([]GetApiKeysResponseApiKey, len(*keys)),
	}

	for i, k := range *keys {
		resp.ApiKeys[i] = GetApiKeysResponseApiKey{
			ID:        k.ID,
			Name:      k.Name,
			CreatedAt: k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return resp
}

type PostApiKeyRequest struct {
	TeamID uint   `param:"teamId" validate:"required,numeric,gte=0"`
	Name   string `json:"name" validate:"required,max=255"`
}

type PostApiKeyResponse struct {
	PlaintextKey string `json:"plaintextKey"`
}

func (h *Handlers) PostApiKey(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostApiKeyRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind PostApiKeyRequest")
		return echo.ErrBadRequest
	}

	plaintextKey, err := h.ApiKeyService.Create(c.Request().Context(), req.TeamID, req.Name)
	if err != nil {
		c.Log.WithError(err).Error("failed to create api key")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusCreated, PostApiKeyResponse{
		PlaintextKey: plaintextKey,
	})
}

type DeleteApiKeyRequest struct {
	TeamID uint `param:"teamId" validate:"required,numeric,gte=0"`
	KeyID  uint `param:"keyId" validate:"required,numeric,gte=0"`
}

func (h *Handlers) DeleteApiKey(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeleteApiKeyRequest](c)
	if err != nil {
		c.Log.WithError(err).Debug("failed to bind DeleteApiKeyRequest")
		return echo.ErrBadRequest
	}

	err = h.ApiKeyService.Delete(c.Request().Context(), req.TeamID, req.KeyID)
	if err != nil {
		c.Log.WithError(err).Error("failed to delete api key")
		return echo.ErrInternalServerError
	}

	return c.NoContent(http.StatusOK)
}

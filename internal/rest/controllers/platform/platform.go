package platform

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/entities"
	platformSvc "github.com/opsway-io/backend/internal/platform"
	hs "github.com/opsway-io/backend/internal/rest/handlers"
	"github.com/opsway-io/backend/internal/rest/helpers"
)

// GetUserProducts returns the resolved product visibility map for the authenticated user.
func (h *Handlers) GetUserProducts(c hs.AuthenticatedContext) error {
	var teamIDPtr *uint
	if rawTeamID := c.QueryParam("teamId"); rawTeamID != "" {
		if parsed, err := strconv.ParseUint(rawTeamID, 10, 64); err == nil && parsed > 0 {
			t := uint(parsed)
			teamIDPtr = &t
		}
	}

	isOwner, products, err := h.PlatformService.GetUserProductVisibility(c.Request().Context(), c.UserID, teamIDPtr)
	if err != nil {
		c.Log.WithError(err).Error("failed to resolve user product visibility")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"isPlatformOwner": isOwner,
		"products":        products,
	})
}

// GetAdminProductFlags returns all product feature flags (platform owner only).
func (h *Handlers) GetAdminProductFlags(c hs.AuthenticatedContext) error {
	flags, err := h.PlatformService.ListFlags(c.Request().Context())
	if err != nil {
		c.Log.WithError(err).Error("failed to list product flags")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"flags": flags,
	})
}

type PutAdminProductFlagRequest struct {
	Product        entities.ProductSlug     `param:"product" validate:"required"`
	Visibility     entities.ProductVisibility `json:"visibility" validate:"required"`
	AllowedUserIDs []uint                     `json:"allowedUserIds"`
	AllowedEmails  []string                   `json:"allowedEmails"`
	AllowedTeamIDs []uint                     `json:"allowedTeamIds"`
	Description    *string                    `json:"description"`
}

// PutAdminProductFlag updates a product's visibility mode and allowlist (platform owner only).
func (h *Handlers) PutAdminProductFlag(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PutAdminProductFlagRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	dto, err := h.PlatformService.UpdateFlag(c.Request().Context(), req.Product, platformSvc.UpdateFlagInput{
		Visibility:     req.Visibility,
		AllowedUserIDs: req.AllowedUserIDs,
		AllowedEmails:  req.AllowedEmails,
		AllowedTeamIDs: req.AllowedTeamIDs,
		Description:    req.Description,
	}, c.UserID)
	if err != nil {
		switch {
		case errors.Is(err, platformSvc.ErrInvalidProduct), errors.Is(err, platformSvc.ErrInvalidVisibility):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		case errors.Is(err, platformSvc.ErrUserNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "One or more specified user emails were not found")
		default:
			c.Log.WithError(err).Error("failed to update product flag")
			return echo.ErrInternalServerError
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"flag": dto,
	})
}

// GetPlatformOwners lists all users with platform owner privileges.
func (h *Handlers) GetPlatformOwners(c hs.AuthenticatedContext) error {
	owners, err := h.PlatformService.ListOwners(c.Request().Context())
	if err != nil {
		c.Log.WithError(err).Error("failed to list platform owners")
		return echo.ErrInternalServerError
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"owners": owners,
	})
}

type PostPlatformOwnerRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// PostPlatformOwner grants platform owner privileges to a user by email.
func (h *Handlers) PostPlatformOwner(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[PostPlatformOwnerRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	owner, err := h.PlatformService.AddOwnerByEmail(c.Request().Context(), req.Email)
	if err != nil {
		if errors.Is(err, platformSvc.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "No user found with that email address")
		}
		c.Log.WithError(err).Error("failed to add platform owner")
		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"owner": owner,
	})
}

type DeletePlatformOwnerRequest struct {
	UserID uint `param:"userId" validate:"required,numeric,gt=0"`
}

// DeletePlatformOwner revokes platform owner privileges from a user.
func (h *Handlers) DeletePlatformOwner(c hs.AuthenticatedContext) error {
	req, err := helpers.Bind[DeletePlatformOwnerRequest](c)
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.PlatformService.RemoveOwner(c.Request().Context(), req.UserID, c.UserID); err != nil {
		switch {
		case errors.Is(err, platformSvc.ErrLastPlatformOwner):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		case errors.Is(err, platformSvc.ErrUserNotFound):
			return echo.ErrNotFound
		default:
			c.Log.WithError(err).Error("failed to remove platform owner")
			return echo.ErrInternalServerError
		}
	}

	return c.NoContent(http.StatusNoContent)
}

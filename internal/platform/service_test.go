package platform

import (
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/stretchr/testify/assert"
)

func TestEvaluateFlag(t *testing.T) {
	team10 := uint(10)
	team99 := uint(99)

	tests := []struct {
		name     string
		flag     *entities.ProductFeatureFlag
		isOwner  bool
		userID   uint
		teamID   *uint
		expected bool
	}{
		{
			name:     "nil flag defaults to enabled",
			flag:     nil,
			isOwner:  false,
			userID:   1,
			expected: true,
		},
		{
			name: "disabled blocks regular user",
			flag: &entities.ProductFeatureFlag{
				Product:    entities.ProductLogSearch,
				Visibility: entities.ProductVisibilityDisabled,
			},
			isOwner:  false,
			userID:   1,
			expected: false,
		},
		{
			name: "disabled blocks platform owner",
			flag: &entities.ProductFeatureFlag{
				Product:    entities.ProductLogSearch,
				Visibility: entities.ProductVisibilityDisabled,
			},
			isOwner:  true,
			userID:   1,
			expected: false,
		},
		{
			name: "owner_only allows platform owner",
			flag: &entities.ProductFeatureFlag{
				Product:    entities.ProductPantomath,
				Visibility: entities.ProductVisibilityOwnerOnly,
			},
			isOwner:  true,
			userID:   1,
			expected: true,
		},
		{
			name: "owner_only blocks non-owner",
			flag: &entities.ProductFeatureFlag{
				Product:    entities.ProductPantomath,
				Visibility: entities.ProductVisibilityOwnerOnly,
			},
			isOwner:  false,
			userID:   2,
			expected: false,
		},
		{
			name: "targeted allows platform owner even if not in allowlist",
			flag: &entities.ProductFeatureFlag{
				Product:        entities.ProductManagedAgent,
				Visibility:     entities.ProductVisibilityTargeted,
				AllowedUserIDs: []uint{42},
			},
			isOwner:  true,
			userID:   1,
			expected: true,
		},
		{
			name: "targeted allows allowlisted user",
			flag: &entities.ProductFeatureFlag{
				Product:        entities.ProductManagedAgent,
				Visibility:     entities.ProductVisibilityTargeted,
				AllowedUserIDs: []uint{42},
			},
			isOwner:  false,
			userID:   42,
			expected: true,
		},
		{
			name: "targeted allows allowlisted team",
			flag: &entities.ProductFeatureFlag{
				Product:        entities.ProductManagedAgent,
				Visibility:     entities.ProductVisibilityTargeted,
				AllowedTeamIDs: []uint{10},
			},
			isOwner:  false,
			userID:   99,
			teamID:   &team10,
			expected: true,
		},
		{
			name: "targeted blocks non-allowlisted user and team",
			flag: &entities.ProductFeatureFlag{
				Product:        entities.ProductManagedAgent,
				Visibility:     entities.ProductVisibilityTargeted,
				AllowedUserIDs: []uint{42},
				AllowedTeamIDs: []uint{10},
			},
			isOwner:  false,
			userID:   99,
			teamID:   &team99,
			expected: false,
		},
		{
			name: "enabled allows everyone",
			flag: &entities.ProductFeatureFlag{
				Product:    entities.ProductMonitoring,
				Visibility: entities.ProductVisibilityEnabled,
			},
			isOwner:  false,
			userID:   99,
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateFlag(tc.flag, tc.isOwner, tc.userID, tc.teamID)
			assert.Equal(t, tc.expected, got)
		})
	}
}

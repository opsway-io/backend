package escalation_test

import (
	"context"
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/escalation"
	"github.com/opsway-io/backend/internal/escalation/mocks"
	"github.com/stretchr/testify/assert"
)

func TestService_GetOnCallUsersByTeamID(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully gets on-call users", func(t *testing.T) {
		repo := new(mocks.Repository)
		svc := escalation.NewService(repo)

		teamID := uint(1)
		tier := 1
		policyID := uint(10)

		policy := &entities.EscalationPolicy{
			ID:     policyID,
			TeamID: teamID,
		}

		rotations := []entities.OnCallRotation{
			{UserID: 100},
			{UserID: 200},
		}

		repo.On("GetPolicyByTeamID", ctx, teamID).Return(policy, nil)
		repo.On("GetRotationsByPolicyIDAndTier", ctx, policyID, tier).Return(rotations, nil)

		users, err := svc.GetOnCallUsersByTeamID(ctx, teamID, tier)

		assert.NoError(t, err)
		assert.ElementsMatch(t, []uint{100, 200}, users)
		repo.AssertExpectations(t)
	})
}

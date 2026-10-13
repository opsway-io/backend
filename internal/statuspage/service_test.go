package statuspage_test

import (
	"context"
	"testing"

	"github.com/opsway-io/backend/internal/entities"
	k8smocks "github.com/opsway-io/backend/internal/k8s/mocks"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/opsway-io/backend/internal/statuspage/mocks"
	"github.com/stretchr/testify/assert"
)

func TestService_Create(t *testing.T) {
	repo := mocks.NewRepository(t)
	k8sSvc := k8smocks.NewService(t)

	svc := statuspage.NewService(repo, k8sSvc)

	ctx := context.Background()
	sp := &entities.StatusPage{
		TeamID: 1,
		Name:   "Test Status Page",
		Domain: "status.example.com",
	}

	repo.On("Create", ctx, sp).Return(nil)
	k8sSvc.On("UpsertIngress", ctx, "status.example.com").Return(nil)

	err := svc.Create(ctx, sp)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	k8sSvc.AssertExpectations(t)
}

func TestService_GetByTeamID(t *testing.T) {
	repo := mocks.NewRepository(t)
	k8sSvc := k8smocks.NewService(t)

	svc := statuspage.NewService(repo, k8sSvc)

	ctx := context.Background()
	expectedPages := []*entities.StatusPage{
		{ID: 1, TeamID: 1, Name: "Page 1"},
		{ID: 2, TeamID: 1, Name: "Page 2"},
	}

	repo.On("GetByTeamID", ctx, uint(1)).Return(expectedPages, nil)

	pages, err := svc.GetByTeamID(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, pages, 2)
	assert.Equal(t, "Page 1", pages[0].Name)
	repo.AssertExpectations(t)
}

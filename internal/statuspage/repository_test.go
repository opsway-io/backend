package statuspage_test

import (
	"context"
	"testing"
	"time"

	_ "github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/statuspage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRepositoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// 1. Start Postgres Container
	dbName := "opsway"
	dbUser := "user"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}()

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// 2. Connect GORM
	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{})
	require.NoError(t, err)

	// Migrate schema
	err = db.AutoMigrate(&entities.Team{}, &entities.StatusPage{}, &entities.StatusPageSubscriber{})
	require.NoError(t, err)

	// 3. Test Repository
	repo := statuspage.NewRepository(db)

	tm := &entities.Team{
		Name: "test-team",
	}
	require.NoError(t, db.Create(tm).Error)

	sp := &entities.StatusPage{
		TeamID: tm.ID,
		Name:   "Acme Status",
		Domain: "status.acme.com",
	}

	// Create
	err = repo.Create(ctx, sp)
	assert.NoError(t, err)
	assert.NotZero(t, sp.ID)

	// Get by ID
	fetched, err := repo.GetByIDAndTeamID(ctx, sp.ID, tm.ID)
	assert.NoError(t, err)
	assert.Equal(t, sp.Name, fetched.Name)
	assert.Equal(t, sp.Domain, fetched.Domain)

	// Update
	sp.Name = "Acme Global Status"
	err = repo.Update(ctx, sp)
	assert.NoError(t, err)

	fetchedUpdate, err := repo.GetByIDAndTeamID(ctx, sp.ID, tm.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Acme Global Status", fetchedUpdate.Name)

	// Subscribers
	sub := &entities.StatusPageSubscriber{
		StatusPageID: sp.ID,
		Email:        "test@example.com",
		Token:        "secret-token",
		Verified:     false,
	}
	err = repo.CreateSubscriber(ctx, sub)
	assert.NoError(t, err)

	err = repo.VerifySubscriber(ctx, "secret-token")
	assert.NoError(t, err)

	subs, err := repo.GetVerifiedSubscribers(ctx, sp.ID)
	assert.NoError(t, err)
	assert.Len(t, subs, 1)
	assert.Equal(t, "test@example.com", subs[0].Email)

	// Delete
	err = repo.Delete(ctx, sp.ID, tm.ID)
	assert.NoError(t, err)

	_, err = repo.GetByIDAndTeamID(ctx, sp.ID, tm.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

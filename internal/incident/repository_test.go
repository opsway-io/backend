package incident_test

import (
	"context"
	"testing"
	"time"

	_ "github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/utils/pointer"
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
	err = db.AutoMigrate(&entities.Team{}, &entities.Monitor{}, &entities.Incident{}, &entities.MonitorAssertion{})
	require.NoError(t, err)

	// 3. Test Repository
	repo := incident.NewRepository(db)

	tm := &entities.Team{
		Name: "test-team",
	}
	require.NoError(t, db.Create(tm).Error)

	inc := &entities.Incident{
		TeamID:      tm.ID,
		Description: pointer.String("API is down"),
		Resolved:    false,
	}

	// Create
	err = repo.Create(ctx, &[]entities.Incident{*inc})
	assert.NoError(t, err)

	var dbInc entities.Incident
	err = db.First(&dbInc, "description = ?", "API is down").Error
	assert.NoError(t, err)
	assert.NotZero(t, dbInc.ID)

	// Get by ID
	fetched, err := repo.GetByID(ctx, dbInc.ID)
	assert.NoError(t, err)
	assert.Equal(t, dbInc.Description, fetched.Description)

	// Update
	fetched.Resolved = true
	err = repo.Update(ctx, fetched)
	assert.NoError(t, err)

	fetchedUpdate, err := repo.GetByID(ctx, dbInc.ID)
	assert.NoError(t, err)
	assert.True(t, fetchedUpdate.Resolved)

	// Delete
	err = repo.Delete(ctx, fetchedUpdate)
	assert.NoError(t, err)

	_, err = repo.GetByID(ctx, dbInc.ID)
	assert.ErrorIs(t, err, incident.ErrNotFound)
}

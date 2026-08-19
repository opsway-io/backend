package maintenance_test

import (
	"context"
	"testing"
	"time"

	_ "github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/maintenance"
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
	err = db.AutoMigrate(&entities.Team{}, &entities.Monitor{}, &entities.Maintenance{}, &entities.MaintenanceSettings{})
	require.NoError(t, err)

	// 3. Test Repository
	repo := maintenance.NewRepository(db)

	tm := &entities.Team{
		Name: "test-team",
	}
	require.NoError(t, db.Create(tm).Error)

	now := time.Now()
	m := &entities.Maintenance{
		TeamID: tm.ID,
		Title:  "Database Upgrade",
		Settings: entities.MaintenanceSettings{
			StartAt: now.Add(-1 * time.Hour), // Started 1 hour ago
			EndAt:   now.Add(1 * time.Hour),  // Ends in 1 hour
		},
	}

	// Create
	err = repo.Create(ctx, m)
	assert.NoError(t, err)
	assert.NotZero(t, m.ID)

	// Get Active
	active, err := repo.GetActive(ctx, now)
	assert.NoError(t, err)
	assert.Len(t, *active, 1)
	assert.Equal(t, "Database Upgrade", (*active)[0].Title)

	// GetActive shouldn't return it if we check for yesterday
	activeYesterday, err := repo.GetActive(ctx, now.Add(-24*time.Hour))
	assert.NoError(t, err)
	assert.Len(t, *activeYesterday, 0)

	// Update
	m.Title = "Database Upgrade V2"
	err = repo.Update(ctx, m)
	assert.NoError(t, err)

	fetchedUpdate, err := repo.GetByID(ctx, m.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Database Upgrade V2", fetchedUpdate.Title)

	// Delete
	err = repo.Delete(ctx, fetchedUpdate)
	assert.NoError(t, err)

	_, err = repo.GetByID(ctx, m.ID)
	assert.ErrorIs(t, err, maintenance.ErrNotFound)
}

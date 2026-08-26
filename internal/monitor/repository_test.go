package monitor_test

import (
	"context"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/monitor"
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
	err = db.AutoMigrate(
		&entities.Team{},
		&entities.Monitor{},
		&entities.MonitorSettings{},
		&entities.MonitorSettingsHeader{},
		&entities.MonitorSettingsBody{},
		&entities.MonitorSettingsTLS{},
		&entities.MonitorAssertion{},
	)
	require.NoError(t, err)

	// Seed data
	team := entities.Team{
		Name: "test-team",
	}
	require.NoError(t, db.Create(&team).Error)

	// 3. Test Repository
	repo := monitor.NewRepository(db)

	m := &entities.Monitor{
		TeamID: team.ID,
		Name:   "Test Monitor",
		State:  entities.MonitorStateActive,
	}

	// Create
	err = repo.Create(ctx, m)
	assert.NoError(t, err)
	assert.NotZero(t, m.ID)

	// Get
	fetched, err := repo.GetMonitorAndSettingsByTeamIDAndID(ctx, team.ID, m.ID)
	assert.NoError(t, err)
	assert.Equal(t, m.Name, fetched.Name)
}

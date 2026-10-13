package team_test

import (
	"context"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/team"
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
	err = db.AutoMigrate(&entities.Team{}, &entities.User{}, &entities.TeamUser{})
	require.NoError(t, err)

	// 3. Test Repository
	repo := team.NewRepository(db)

	u := &entities.User{
		Name:  "Test User",
		Email: "test@opsway.eu",
	}
	require.NoError(t, db.Create(u).Error)

	tm := &entities.Team{
		Name: "test-team",
	}

	// Create with Owner
	err = repo.CreateWithOwnerUserID(ctx, tm, u.ID)
	require.NoError(t, err)
	assert.NotZero(t, tm.ID)

	// Get
	fetched, err := repo.GetByID(ctx, tm.ID)
	require.NoError(t, err)
	assert.Equal(t, tm.Name, fetched.Name)
}

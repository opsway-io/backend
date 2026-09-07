package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/incident"
	"gorm.io/gorm"
)

func TestGormUpdates(t *testing.T) {
	// Need to connect to DB to test this. But we don't have DB credentials.
	// Wait, we can look at GORM documentation.
}

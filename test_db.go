package main

import (
	"context"
	"fmt"
	"log"

	"github.com/opsway-io/backend/internal/incident"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=opsway password=opsway dbname=opsway port=5432 sslmode=disable"
	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	repo := incident.NewRepository(db)
	limit := 10
	offset := 0
	incidents, err := repo.GetByMonitorIDWithAssertionPaginated(context.Background(), 1, &offset, &limit)
	if err != nil {
		log.Fatal(err)
	}
	if len(*incidents) == 0 {
		fmt.Println("No incidents found for monitor 1. Trying to find any incidents...")
		var all []incident.IncidentAndAssertion
		db.Table("incidents").Select("incidents.*").Find(&all)
		incidents = &all
	}

	for _, inc := range *incidents {
		fmt.Printf("ID: %d, Acknowledged: %v, AcknowledgedBy: %v, AcknowledgedAt: %v, Property: %v\n", 
			inc.ID, inc.Acknowledged, inc.AcknowledgedBy, inc.AcknowledgedAt, inc.Property)
	}
}

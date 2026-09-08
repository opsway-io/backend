package main

import (
	"context"
	"fmt"
	"log"

	"github.com/opsway-io/backend/internal/team"
	"gorm.io/gorm"
	pg "gorm.io/driver/postgres"
)

func main() {
	dsn := "host=localhost user=postgres password=pass dbname=opsway port=5432 sslmode=disable"
	db, err := gorm.Open(pg.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	repo := team.NewRepository(db)
	offset, limit := 0, 10
	q := ""
	
	users, err := repo.GetUsersByID(context.Background(), 1, &offset, &limit, &q, nil)
	if err != nil {
		log.Fatal(err)
	}

	for _, u := range *users {
		fmt.Printf("User ID: %d, Email: %s\n", u.ID, u.Email)
	}
}

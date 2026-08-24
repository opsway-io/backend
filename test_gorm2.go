package main

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID           uint
	Email        string
	PasswordHash *string
}

func main() {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&User{})

	oldHash := "old_hash"
	u := &User{Email: "test@example.com", PasswordHash: &oldHash}
	db.Create(u)

	var fetched User
	db.First(&fetched, u.ID)

	newHash := "new_hash"
	fetched.PasswordHash = &newHash
	
	db.Model(&fetched).Updates(&fetched)

	var verify User
	db.First(&verify, u.ID)
	fmt.Printf("PasswordHash updated: %v (expected true)\n", *verify.PasswordHash == "new_hash")
}

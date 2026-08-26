package main

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/utils/pointer"
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

	u := &User{Email: "test@example.com", PasswordHash: pointer.String("old_hash")}
	db.Create(u)

	// Fetch user
	var fetched User
	db.First(&fetched, u.ID)

	// Change password
	fetched.PasswordHash = pointer.String("new_hash")

	// Update user
	db.Model(&fetched).Updates(&fetched)

	// Verify
	var verify User
	db.First(&verify, u.ID)
	fmt.Printf("PasswordHash: %v\n", *verify.PasswordHash)
}

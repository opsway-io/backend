package main

import (
	"context"
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
	db, _ := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	db.AutoMigrate(&User{})

	oldHash := "old_hash"
	u := &User{Email: "test@example.com", PasswordHash: &oldHash}
	db.Create(u)

	var user User
	err := db.WithContext(context.Background()).Where(User{Email: "test@example.com"}).First(&user).Error
	if err != nil {
		panic(err)
	}
	
	if user.PasswordHash == nil {
		fmt.Println("PasswordHash is nil!")
	} else {
		fmt.Println("PasswordHash is loaded correctly!")
	}
}

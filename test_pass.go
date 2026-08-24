package main

import (
	"fmt"
	"github.com/opsway-io/backend/internal/entities"
)

func main() {
	u := &entities.User{}
	pw := "correcthorsebatterystaple"
	err := u.SetPassword(pw)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	ok := u.CheckPassword(pw)
	fmt.Printf("Matches original: %v\n", ok)
}

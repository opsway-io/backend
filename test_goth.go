package main

import (
	"fmt"
	"github.com/markbates/goth/providers/google"
)

func main() {
	p := google.New("clientid", "secret", "http://localhost/callback", "email", "profile")
	fmt.Printf("%+v\n", p)
}

package utils

import (
	"encoding/json"
	"fmt"
	"log"
)

func Print(v interface{}) {
	v, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("%s\n", v)
}

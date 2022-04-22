package main

import (
	"fmt"
	"net/http"
	"time"

	"monitor/internal/checker"
)

func main() {
	r, _ := checker.APICheck(http.MethodGet, "http://example.com", nil, nil, time.Second)
	fmt.Printf("%s/n", r.Body)
	// cmd.Execute()
}

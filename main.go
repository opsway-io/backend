package main

import "github.com/opsway-io/backend/cmd"

const (
	subject        = "tickets"
	consumersGroup = "tickets-consumer-group"
)

func main() {
	cmd.Execute()
}

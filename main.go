package main

import "monitor/cmd"

const (
	subject        = "tickets"
	consumersGroup = "tickets-consumer-group"
)

func main() {
	cmd.Execute()
}

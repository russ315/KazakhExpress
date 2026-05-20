package main

import (
	"log"

	"kazakhexpress/api-gateway/internal/gatewayapp"
)

func main() {
	if err := gatewayapp.Run(); err != nil {
		log.Fatal(err)
	}
}

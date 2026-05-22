package main

import (
	"context"
	"log"

	"kazakhexpress/review-service/internal/reviewapp"
)

func main() {
	if err := reviewapp.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

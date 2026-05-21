package main

import (
	"context"
	"log"

	"kazakhexpress/payment-service/internal/paymentapp"
)

func main() {
	if err := paymentapp.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"
	"log"

	"kazakhexpress/smtp-service/internal/smtpapp"
)

func main() {
	if err := smtpapp.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

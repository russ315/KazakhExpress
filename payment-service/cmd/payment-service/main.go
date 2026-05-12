package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"kazakhexpress/payment-service/internal/email"
	httpapi "kazakhexpress/payment-service/internal/http"
	"kazakhexpress/payment-service/internal/messaging"
	"kazakhexpress/payment-service/internal/payment"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

func main() {
	ctx := context.Background()
	port := getEnv("PAYMENT_SERVICE_PORT", "8083")

	db, err := pgxpool.New(ctx, getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	natsConn, err := nats.Connect(getEnv("NATS_URL", nats.DefaultURL))
	if err != nil {
		log.Fatal(err)
	}
	defer natsConn.Close()

	repo := payment.NewPostgresRepository(db)
	publisher := messaging.NewNATSPublisher(natsConn)
	emailer := email.NewSMTPSender(email.SMTPConfig{
		Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		Port:     getEnv("SMTP_PORT", "587"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     getEnv("SMTP_FROM", "noreply@kazakhexpress.kz"),
	})
	service := payment.NewService(repo, publisher, emailer)
	handler := httpapi.NewHandler(service)

	log.Printf("payment service started on :%s", port)
	if err := http.ListenAndServe(":"+port, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

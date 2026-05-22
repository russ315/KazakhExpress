package main

import (
	"context"
	"log"
	"net/http"
	"os"

	httpapi "kazakhexpress/order-service/internal/http"
	"kazakhexpress/order-service/internal/order"
	"kazakhexpress/order-service/internal/rabbitmq"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	port := getEnv("ORDER_SERVICE_PORT", "8080")

	db, err := pgxpool.New(ctx, getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := order.NewPostgresRepository(db)
	var publisher order.EventPublisher = order.NoopPublisher{}
	if rmq, err := rabbitmq.NewPublisher(getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")); err != nil {
		log.Printf("warning: rabbitmq not available: %v", err)
	} else {
		publisher = rmq
	}
	service := order.NewService(repo, publisher)
	handler := httpapi.NewHandler(service)

	log.Printf("order service started on :%s", port)
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

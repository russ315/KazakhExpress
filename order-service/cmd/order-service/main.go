package main

import (
	"log"
	"net/http"
	"os"

	httpapi "kazakhexpress/order-service/internal/http"
	"kazakhexpress/order-service/internal/order"
)

func main() {
	port := os.Getenv("ORDER_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	repo := order.NewMemoryRepository()
	service := order.NewService(repo)
	handler := httpapi.NewHandler(service)

	log.Printf("order service started on :%s", port)
	if err := http.ListenAndServe(":"+port, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}

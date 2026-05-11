package main

import (
	"log"
	"net/http"
	"os"

	httpapi "kazakhexpress/product-service/internal/http"
	"kazakhexpress/product-service/internal/product"
)

func main() {
	port := os.Getenv("PRODUCT_SERVICE_PORT")
	if port == "" {
		port = "8082"
	}

	repo := product.NewMemoryRepository()
	service := product.NewService(repo)
	handler := httpapi.NewHandler(service)

	log.Printf("product service started on :%s", port)
	if err := http.ListenAndServe(":"+port, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}

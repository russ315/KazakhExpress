package main

import (
	"log"
	"net/http"
	"os"

	httpapi "kazakhexpress/product-service/internal/http"
	"kazakhexpress/product-service/internal/product"
	"kazakhexpress/product-service/internal/rabbitmq"
	redisclient "kazakhexpress/product-service/internal/redis"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	port := getEnv("PRODUCT_SERVICE_PORT", "8082")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	repo := product.NewMemoryRepository()

	var cache product.CacheService = product.NoopCache{}
	redisClient, err := redisclient.NewClient(redisAddr)
	if err != nil {
		log.Printf("warning: redis not available: %v", err)
	} else {
		cache = redisclient.NewCacheAdapter(redisClient)
		defer redisClient.Close()
	}

	var publisher product.EventPublisher = product.NoopPublisher{}
	rmqPublisher, err := rabbitmq.NewPublisher(rabbitURL)
	if err != nil {
		log.Printf("warning: rabbitmq not available: %v", err)
	} else {
		publisher = rabbitmq.NewEventAdapter(rmqPublisher)
		defer rmqPublisher.Close()
	}

	service := product.NewService(repo, cache, publisher)
	handler := httpapi.NewHandler(service)

	log.Printf("product service started on :%s", port)
	if err := http.ListenAndServe(":"+port, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}

package gatewayapp

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"kazakhexpress/api-gateway/internal/gateway"
	"kazakhexpress/api-gateway/internal/orderservice"
	"kazakhexpress/api-gateway/internal/paymentservice"
	"kazakhexpress/api-gateway/internal/productservice"
	"kazakhexpress/api-gateway/internal/reviewservice"
	"kazakhexpress/api-gateway/internal/userservice"

	"github.com/redis/go-redis/v9"
)

func Run() error {
	port := getEnv("API_GATEWAY_PORT", "8080")
	routerOptions := []gateway.RouterOption{}
	if redisAddr := getEnv("REDIS_ADDR", ""); redisAddr != "" {
		redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
		defer redisClient.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Printf("gateway redis rate limiter disabled: %v", err)
		} else {
			limit := getEnvInt("RATE_LIMIT_REQUESTS", 120)
			window := time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second
			routerOptions = append(routerOptions, gateway.WithRateLimiter(gateway.NewRedisRateLimiter(redisClient), limit, window))
			log.Printf("gateway redis rate limiter enabled: %d requests per %s", limit, window)
		}
		cancel()
	}
	router := gateway.NewRouter(routerOptions...)

	paymentClient, err := paymentservice.NewGRPCClient(getEnv("PAYMENT_GRPC_ADDR", "localhost:9093"))
	if err != nil {
		return err
	}
	defer paymentClient.Close()
	paymentservice.RegisterRoutes(router, paymentClient)

	userClient, err := userservice.NewGRPCClient(getEnv("USER_GRPC_ADDR", "localhost:50051"))
	if err != nil {
		return err
	}
	defer userClient.Close()
	userservice.RegisterRoutes(router, userClient)

	orderClient, err := orderservice.NewGRPCClient(getEnv("ORDER_GRPC_ADDR", "localhost:9092"))
	if err != nil {
		return err
	}
	defer orderClient.Close()
	orderservice.RegisterRoutes(router, orderClient)

	productClient, err := productservice.NewGRPCClient(getEnv("PRODUCT_GRPC_ADDR", "localhost:9095"))
	if err != nil {
		return err
	}
	defer productClient.Close()
	productservice.RegisterRoutes(router, productClient)

	reviewClient, err := reviewservice.NewGRPCClient(getEnv("REVIEW_GRPC_ADDR", "localhost:9096"))
	if err != nil {
		return err
	}
	defer reviewClient.Close()
	reviewservice.RegisterRoutes(router, reviewClient)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("api gateway started on :%s", port)
	return server.ListenAndServe()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

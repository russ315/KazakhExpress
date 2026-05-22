package gatewayapp

import (
	"log"
	"net/http"
	"os"
	"time"

	"kazakhexpress/api-gateway/internal/gateway"
	"kazakhexpress/api-gateway/internal/orderservice"
	"kazakhexpress/api-gateway/internal/paymentservice"
	"kazakhexpress/api-gateway/internal/userservice"
)

func Run() error {
	port := getEnv("API_GATEWAY_PORT", "8080")
	router := gateway.NewRouter()

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

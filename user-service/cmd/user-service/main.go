package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"kazakhexpress/user-service/internal/email"
	grpcapi "kazakhexpress/user-service/internal/grpc"
	httpapi "kazakhexpress/user-service/internal/http"
	"kazakhexpress/user-service/internal/rabbitmq"
	redisclient "kazakhexpress/user-service/internal/redis"
	"kazakhexpress/user-service/internal/user"

	userv1 "github.com/russ315/kazakhexpress-protos/kazakhexpress/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	port := getEnv("USER_SERVICE_PORT", "8081")
	grpcPort := getEnv("USER_SERVICE_GRPC_PORT", "50051")

	dbURL := getEnv("DATABASE_URL", "postgresql://postgres:Ruslan2006%40@localhost:5432/kazakhexpress_users?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-in-production")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	repo, err := user.NewPostgresRepository(dbURL)
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	smtpEmailService := email.NewSMTPEmailService()

	var eventSvc user.EventService
	rmqPublisher, err := rabbitmq.NewPublisher(rabbitURL)
	if err != nil {
		log.Printf("Warning: RabbitMQ not available: %v", err)
	} else {
		eventSvc = user.NewRabbitMQEventAdapter(rmqPublisher)
		defer rmqPublisher.Close()
	}

	var cacheSvc user.CacheService
	var rateLimitSvc user.RateLimitService
	redisClient, err := redisclient.NewClient(redisAddr)
	if err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	} else {
		cacheSvc = user.NewRedisCacheAdapter(redisClient)
		rateLimitSvc = user.NewRedisRateLimitAdapter(redisClient)
		defer redisClient.Close()
	}

	svc := user.NewService(repo, jwtSecret, smtpEmailService, eventSvc, cacheSvc, rateLimitSvc)

	go startGRPCServer(grpcPort, svc)

	httpHandler := httpapi.NewHandler(svc)
	log.Printf("user service HTTP started on :%s, gRPC on :%s", port, grpcPort)
	if err := http.ListenAndServe(":"+port, httpHandler.Routes()); err != nil {
		log.Fatal(err)
	}
}

func startGRPCServer(port string, svc user.Service) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	grpcServer := grpc.NewServer()
	grpcHandler := grpcapi.NewUserGRPCHandler(svc)
	userv1.RegisterUserServiceServer(grpcServer, grpcHandler)

	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

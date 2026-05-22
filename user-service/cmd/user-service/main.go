package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"kazakhexpress/user-service/internal/email"
	grpcapi "kazakhexpress/user-service/internal/grpc"
	httpapi "kazakhexpress/user-service/internal/http"
	"kazakhexpress/user-service/internal/messaging"
	redisclient "kazakhexpress/user-service/internal/redis"
	"kazakhexpress/user-service/internal/user"

	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	userv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/user/v1"
	"github.com/nats-io/nats.go"

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
	natsURL := getEnv("NATS_URL", nats.DefaultURL)

	repo, err := user.NewPostgresRepository(dbURL)
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	smtpEmailService, err := email.NewGRPCEmailService(getEnv("SMTP_GRPC_ADDR", "localhost:9094"))
	if err != nil {
		log.Printf("Warning: SMTP service unavailable: %v", err)
	}
	if smtpEmailService != nil {
		defer smtpEmailService.Close()
	}

	var eventSvc user.EventService
	natsConn, err := nats.Connect(natsURL)
	if err != nil {
		log.Printf("Warning: NATS not available: %v", err)
	} else {
		eventSvc = user.NewNATSEventAdapter(messaging.NewPublisher(natsConn))
		defer natsConn.Close()
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

	grpcprom.DefaultServerMetrics.EnableHandlingTimeHistogram()

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcprom.DefaultServerMetrics.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpcprom.DefaultServerMetrics.StreamServerInterceptor()),
	)
	grpcHandler := grpcapi.NewUserGRPCHandler(svc)
	userv1.RegisterUserServiceServer(grpcServer, grpcHandler)
	grpcprom.DefaultServerMetrics.InitializeMetrics(grpcServer)

	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

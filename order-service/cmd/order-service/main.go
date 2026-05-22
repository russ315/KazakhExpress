package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"kazakhexpress/order-service/internal/cache"
	"kazakhexpress/order-service/internal/grpcapi"
	httpapi "kazakhexpress/order-service/internal/http"
	"kazakhexpress/order-service/internal/messaging"
	"kazakhexpress/order-service/internal/order"

	"github.com/jackc/pgx/v5/pgxpool"
	orderv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/order/v1"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	port := getEnv("ORDER_SERVICE_PORT", "8080")
	grpcPort := getEnv("ORDER_GRPC_PORT", "9092")

	db, err := pgxpool.New(ctx, getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: getEnv("REDIS_ADDR", "localhost:6379")})
	defer redisClient.Close()

	publisher, consumer, cleanupMessaging := mustNATS()
	defer cleanupMessaging()

	service := order.NewService(
		order.NewPostgresRepository(db),
		publisher,
		cache.NewRedisStatusCache(redisClient, 24*time.Hour),
	)
	if consumer != nil {
		natsConsumer := consumer(service)
		if natsConsumer != nil {
			defer natsConsumer.Close()
			if err := natsConsumer.Start(ctx); err != nil {
				log.Fatal(err)
			}
		}
	}

	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, grpcapi.NewServer(service))
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Printf("order grpc started on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("order grpc stopped: %v", err)
		}
	}()
	defer grpcServer.GracefulStop()

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

func mustNATS() (order.EventPublisher, func(*order.Service) *messaging.NATSConsumer, func()) {
	conn, err := nats.Connect(getEnv("NATS_URL", nats.DefaultURL))
	if err != nil {
		log.Printf("nats disabled: %v", err)
		return nil, nil, func() {}
	}
	return messaging.NewNATSPublisher(conn), func(service *order.Service) *messaging.NATSConsumer {
		return messaging.NewNATSConsumer(conn, service)
	}, conn.Close
}

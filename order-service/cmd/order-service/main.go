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
	amqp "github.com/rabbitmq/amqp091-go"
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

	publisher, consumer, cleanupMessaging := mustRabbit(ctx)
	defer cleanupMessaging()

	service := order.NewService(
		order.NewPostgresRepository(db),
		publisher,
		cache.NewRedisStatusCache(redisClient, 24*time.Hour),
	)
	if consumer != nil {
		rabbitConsumer := consumer(service)
		if rabbitConsumer != nil {
			if err := rabbitConsumer.Start(ctx); err != nil {
				log.Fatal(err)
			}
		}
	}

	grpcServer := grpc.NewServer()
	grpcapi.Register(grpcServer, grpcapi.NewServer(service))
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

func mustRabbit(context.Context) (order.EventPublisher, func(*order.Service) *messaging.RabbitConsumer, func()) {
	conn, err := amqp.Dial(getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"))
	if err != nil {
		log.Printf("rabbitmq disabled: %v", err)
		return nil, nil, func() {}
	}
	pubChannel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		log.Printf("rabbitmq publisher disabled: %v", err)
		return nil, nil, func() {}
	}
	publisher, err := messaging.NewRabbitPublisher(pubChannel, getEnv("RABBITMQ_EXCHANGE", "kazakhexpress.events"))
	if err != nil {
		_ = pubChannel.Close()
		_ = conn.Close()
		log.Printf("rabbitmq publisher disabled: %v", err)
		return nil, nil, func() {}
	}

	consumerFactory := func(service *order.Service) *messaging.RabbitConsumer {
		consumeChannel, err := conn.Channel()
		if err != nil {
			log.Printf("rabbitmq consumer disabled: %v", err)
			return nil
		}
		consumer, err := messaging.NewRabbitConsumer(consumeChannel, getEnv("RABBITMQ_EXCHANGE", "kazakhexpress.events"), service)
		if err != nil {
			_ = consumeChannel.Close()
			log.Printf("rabbitmq consumer disabled: %v", err)
			return nil
		}
		return consumer
	}

	return publisher, consumerFactory, func() {
		_ = pubChannel.Close()
		_ = conn.Close()
	}
}

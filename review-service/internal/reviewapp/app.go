package reviewapp

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"time"

	"kazakhexpress/review-service/internal/grpcapi"
	"kazakhexpress/review-service/internal/rabbitmq"
	redisclient "kazakhexpress/review-service/internal/redis"
	"kazakhexpress/review-service/internal/review"
	reviewv1 "kazakhexpress/review-service/internal/reviewv1"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {
	grpcPort := getEnv("REVIEW_GRPC_PORT", "9095")
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kazakhexpress_reviews?sslmode=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := retry(ctx, "postgres", func() error { return db.Ping(ctx) }); err != nil {
		return err
	}

	var cache review.CacheService = review.NoopCache{}
	redisClient, err := redisclient.NewClient(redisAddr)
	if err != nil {
		log.Printf("warning: redis not available: %v", err)
	} else {
		cache = redisclient.NewCacheAdapter(redisClient)
		defer redisClient.Close()
	}

	var publisher review.EventPublisher = review.NoopPublisher{}
	amqpConn, err := dialRabbit(ctx, rabbitURL)
	if err != nil {
		log.Printf("warning: rabbitmq not available: %v", err)
	} else {
		defer amqpConn.Close()
		rmqPublisher, err := rabbitmq.NewPublisher(amqpConn)
		if err != nil {
			return err
		}
		defer rmqPublisher.Close()
		publisher = rabbitmq.NewEventAdapter(rmqPublisher)

	}

	repo := review.NewPostgresRepository(db)
	service := review.NewService(repo, cache, publisher)
	if amqpConn != nil {
		consumer := rabbitmq.NewConsumer(service)
		if err := consumer.Start(ctx, amqpConn); err != nil {
			log.Printf("warning: order.completed consumer failed: %v", err)
		}
	}
	return startGRPC(ctx, grpcPort, service)
}

func dialRabbit(ctx context.Context, url string) (*amqp.Connection, error) {
	var conn *amqp.Connection
	err := retry(ctx, "rabbitmq", func() error {
		var err error
		conn, err = amqp.Dial(url)
		return err
	})
	return conn, err
}

func startGRPC(ctx context.Context, port string, service *review.Service) error {
	grpcServer := grpc.NewServer()
	reviewv1.RegisterReviewServiceServer(grpcServer, grpcapi.NewServer(service))

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	log.Printf("review grpc started on :%s", port)
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

func retry(ctx context.Context, name string, operation func() error) error {
	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		if err := operation(); err != nil {
			lastErr = err
			log.Printf("%s is not ready yet, retry %d/30: %v", name, attempt, err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

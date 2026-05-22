package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"kazakhexpress/review-service/internal/cache"
	"kazakhexpress/review-service/internal/grpcapi"
	"kazakhexpress/review-service/internal/messaging"
	"kazakhexpress/review-service/internal/review"

	"github.com/jackc/pgx/v5/pgxpool"
	reviewv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/review/v1"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	grpcPort := getEnv("REVIEW_GRPC_PORT", "9096")

	db, err := pgxpool.New(ctx, getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := retry(ctx, "postgres", func() error { return db.Ping(ctx) }); err != nil {
		log.Fatal(err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: getEnv("REDIS_ADDR", "localhost:6379")})
	defer redisClient.Close()

	var publisher review.EventPublisher
	nc, err := nats.Connect(getEnv("NATS_URL", nats.DefaultURL))
	if err != nil {
		log.Printf("nats disabled: %v", err)
	} else {
		defer nc.Close()
		publisher = messaging.NewNATSPublisher(nc)
	}

	service := review.NewService(review.NewPostgresRepository(db), cache.NewRedisRatingCache(redisClient, time.Hour), publisher)
	server := grpc.NewServer()
	reviewv1.RegisterReviewServiceServer(server, grpcapi.NewServer(service))
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("review grpc started on :%s", grpcPort)
	if err := server.Serve(listener); err != nil {
		log.Fatal(err)
	}
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

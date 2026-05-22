package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"kazakhexpress/product-service/internal/grpcapi"
	httpapi "kazakhexpress/product-service/internal/http"
	"kazakhexpress/product-service/internal/messaging"
	"kazakhexpress/product-service/internal/product"
	"kazakhexpress/product-service/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	productv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/product/v1"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	httpPort := getEnv("PRODUCT_SERVICE_PORT", "8084")
	grpcPort := getEnv("PRODUCT_GRPC_PORT", "9095")

	db, err := pgxpool.New(ctx, getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := retry(ctx, "postgres", func() error { return db.Ping(ctx) }); err != nil {
		log.Fatal(err)
	}

	minioStore, err := storage.NewMinIO(
		getEnv("MINIO_ENDPOINT", "localhost:9000"),
		getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		getEnv("MINIO_SECRET_KEY", "minioadmin"),
		getEnv("MINIO_BUCKET", "products"),
		getEnv("MINIO_USE_SSL", "false") == "true",
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := retry(ctx, "minio", func() error { return minioStore.EnsureBucket(ctx) }); err != nil {
		log.Fatal(err)
	}

	var publisher product.EventPublisher
	nc, err := nats.Connect(getEnv("NATS_URL", nats.DefaultURL))
	if err != nil {
		log.Printf("nats disabled: %v", err)
	} else {
		defer nc.Close()
		publisher = messaging.NewNATSPublisher(nc)
	}

	service := product.NewService(product.NewPostgresRepository(db), minioStore, publisher)

	grpcServer := grpc.NewServer()
	productv1.RegisterProductServiceServer(grpcServer, grpcapi.NewServer(service))
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Printf("product grpc started on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("product grpc stopped: %v", err)
		}
	}()
	defer grpcServer.GracefulStop()

	log.Printf("product http started on :%s", httpPort)
	if err := http.ListenAndServe(":"+httpPort, httpapi.NewHandler(service).Routes()); err != nil {
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

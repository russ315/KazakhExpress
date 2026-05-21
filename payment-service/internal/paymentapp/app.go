package paymentapp

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"kazakhexpress/payment-service/internal/cache"
	"kazakhexpress/payment-service/internal/email"
	"kazakhexpress/payment-service/internal/grpcapi"
	httpapi "kazakhexpress/payment-service/internal/http"
	"kazakhexpress/payment-service/internal/messaging"
	"kazakhexpress/payment-service/internal/payment"
	"kazakhexpress/payment-service/internal/provider"

	"github.com/jackc/pgx/v5/pgxpool"
	paymentv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/payment/v1"
	"github.com/nats-io/nats.go"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func Run(ctx context.Context) error {
	httpPort := getEnv("PAYMENT_SERVICE_PORT", "8083")
	grpcPort := getEnv("PAYMENT_GRPC_PORT", "9093")

	db, err := pgxpool.New(ctx, getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kazakhexpress?sslmode=disable"))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := retry(ctx, "postgres", func() error {
		return db.Ping(ctx)
	}); err != nil {
		return err
	}

	publisher, cleanupPublisher, err := newPublisher(ctx)
	if err != nil {
		return err
	}
	defer cleanupPublisher()

	redisClient := redis.NewClient(&redis.Options{Addr: getEnv("REDIS_ADDR", "localhost:6379")})
	if err := retry(ctx, "redis", func() error {
		return redisClient.Ping(ctx).Err()
	}); err != nil {
		return err
	}
	defer redisClient.Close()

	service := payment.NewService(
		payment.NewPostgresRepository(db),
		publisher,
		mustEmailSender(),
		cache.NewRedisIdempotencyStore(redisClient, 24*time.Hour),
		provider.NewMockProvider(),
	)

	grpcServer := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(grpcServer, grpcapi.NewServer(service))
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return err
	}
	go func() {
		log.Printf("payment grpc started on :%s", grpcPort)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("payment grpc stopped: %v", err)
		}
	}()
	defer grpcServer.GracefulStop()

	httpServer := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           httpapi.NewHandler(service).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("payment http started on :%s", httpPort)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func mustEmailSender() payment.EmailSender {
	sender, err := email.NewGRPCSender(getEnv("SMTP_GRPC_ADDR", "localhost:9094"))
	if err != nil {
		log.Fatal(err)
	}
	return sender
}

func newPublisher(ctx context.Context) (payment.EventPublisher, func(), error) {
	if getEnv("MESSAGE_BROKER", "rabbitmq") == "nats" {
		var natsConn *nats.Conn
		err := retry(ctx, "nats", func() error {
			var err error
			natsConn, err = nats.Connect(getEnv("NATS_URL", nats.DefaultURL))
			return err
		})
		if err != nil {
			return nil, nil, err
		}
		return messaging.NewNATSPublisher(natsConn), natsConn.Close, nil
	}

	var (
		conn      *amqp.Connection
		channel   *amqp.Channel
		publisher *messaging.RabbitPublisher
	)
	err := retry(ctx, "rabbitmq", func() error {
		var err error
		conn, err = amqp.Dial(getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"))
		if err != nil {
			return err
		}
		channel, err = conn.Channel()
		if err != nil {
			_ = conn.Close()
			return err
		}
		publisher, err = messaging.NewRabbitPublisher(channel, getEnv("RABBITMQ_EXCHANGE", "kazakhexpress.events"))
		if err != nil {
			_ = channel.Close()
			_ = conn.Close()
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return publisher, func() {
		_ = channel.Close()
		_ = conn.Close()
	}, nil
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

package smtpapp

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	grpcprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	smtpv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/smtp/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"kazakhexpress/smtp-service/internal/grpcapi"
	smtpservice "kazakhexpress/smtp-service/internal/smtp"
)

func Run(ctx context.Context) error {
	port := getEnv("SMTP_GRPC_PORT", "9094")
	metricsPort := getEnv("SMTP_METRICS_PORT", "9104")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	grpcprom.DefaultServerMetrics.EnableHandlingTimeHistogram()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpcprom.DefaultServerMetrics.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpcprom.DefaultServerMetrics.StreamServerInterceptor()),
	)
	smtpv1.RegisterSMTPServiceServer(server, grpcapi.NewServer(smtpservice.NewService(smtpservice.Config{
		Host:         getEnv("SMTP_HOST", "smtp.gmail.com"),
		Port:         getEnv("SMTP_PORT", "587"),
		Username:     os.Getenv("SMTP_USERNAME"),
		Password:     os.Getenv("SMTP_PASSWORD"),
		From:         getEnv("SMTP_FROM", "noreply@kazakhexpress.kz"),
		ResendAPIKey: os.Getenv("RESEND_API_KEY"),
		ResendFrom:   getEnv("RESEND_FROM", "KazakhExpress <noreply@send.maqsatto.dev>"),
	})))
	grpcprom.DefaultServerMetrics.InitializeMetrics(server)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Printf("smtp metrics started on :%s", metricsPort)
		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil {
			log.Printf("smtp metrics stopped: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	log.Printf("smtp grpc started on :%s", port)
	return server.Serve(listener)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

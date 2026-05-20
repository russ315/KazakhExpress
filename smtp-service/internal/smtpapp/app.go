package smtpapp

import (
	"context"
	"log"
	"net"
	"os"

	smtpv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/smtp/v1"
	"google.golang.org/grpc"
	"kazakhexpress/smtp-service/internal/grpcapi"
	smtpservice "kazakhexpress/smtp-service/internal/smtp"
)

func Run(ctx context.Context) error {
	port := getEnv("SMTP_GRPC_PORT", "9094")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	smtpv1.RegisterSMTPServiceServer(server, grpcapi.NewServer(smtpservice.NewService(smtpservice.Config{
		Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		Port:     getEnv("SMTP_PORT", "587"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     getEnv("SMTP_FROM", "noreply@kazakhexpress.kz"),
	})))

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

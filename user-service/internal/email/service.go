package email

import (
	"context"
	"fmt"
	"time"

	smtpv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/smtp/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Service interface {
	SendWelcomeEmail(email, firstName string) error
}

type GRPCEmailService struct {
	conn   *grpc.ClientConn
	client smtpv1.SMTPServiceClient
}

func NewGRPCEmailService(target string) (*GRPCEmailService, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create smtp grpc client: %w", err)
	}
	return &GRPCEmailService{conn: conn, client: smtpv1.NewSMTPServiceClient(conn)}, nil
}

func (s *GRPCEmailService) Close() error {
	return s.conn.Close()
}

func (s *GRPCEmailService) SendWelcomeEmail(email, firstName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.client.SendWelcomeEmail(ctx, &smtpv1.WelcomeEmailRequest{To: email, FirstName: firstName})
	return err
}

package smtp

import (
	"context"
	"fmt"
	"log"
	netsmtp "net/smtp"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Service struct {
	config Config
}

func NewService(config Config) *Service {
	return &Service{config: config}
}

func (s *Service) SendEmail(ctx context.Context, to string, subject string, body string) error {
	if to == "" || subject == "" || body == "" {
		return fmt.Errorf("email to, subject and body are required")
	}
	if s.config.Username == "" || s.config.Password == "" {
		log.Printf("smtp dry-run to=%s subject=%q", to, subject)
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	auth := netsmtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.config.From, to, subject, body)
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	if err := netsmtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("send smtp email: %w", err)
	}
	return nil
}

func WelcomeSubject() string {
	return "Welcome to KazakhExpress"
}

func WelcomeBody(firstName string) string {
	if firstName == "" {
		firstName = "there"
	}
	return fmt.Sprintf("Hi %s, welcome to KazakhExpress.", firstName)
}

func PaymentReceiptSubject() string {
	return "KazakhExpress payment receipt"
}

func PaymentReceiptBody(paymentID string, orderID string, amountKZT int64) string {
	return fmt.Sprintf("Payment %s for order %s was paid successfully. Amount: %d KZT.", paymentID, orderID, amountKZT)
}

func PaymentRefundSubject() string {
	return "KazakhExpress payment refund"
}

func PaymentRefundBody(paymentID string, orderID string, amountKZT int64, reason string) string {
	return fmt.Sprintf("Payment %s for order %s was refunded. Amount: %d KZT. Reason: %s.", paymentID, orderID, amountKZT, reason)
}

func PaymentFailureSubject() string {
	return "KazakhExpress payment failed"
}

func PaymentFailureBody(paymentID string, orderID string, amountKZT int64, reason string) string {
	return fmt.Sprintf("Payment %s for order %s failed. Amount: %d KZT. Reason: %s.", paymentID, orderID, amountKZT, reason)
}

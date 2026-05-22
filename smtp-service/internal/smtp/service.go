package smtp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	netsmtp "net/smtp"
)

type Config struct {
	Host         string
	Port         string
	Username     string
	Password     string
	From         string
	ResendAPIKey string
	ResendFrom   string
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
	if s.config.ResendAPIKey != "" {
		return s.sendResend(ctx, to, subject, body)
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

func (s *Service) sendResend(ctx context.Context, to string, subject string, body string) error {
	from := s.config.ResendFrom
	if from == "" {
		from = s.config.From
	}
	payload := map[string]any{
		"from":    from,
		"to":      []string{to},
		"subject": subject,
		"text":    body,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend email: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.config.ResendAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send resend email: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("resend email rejected: status=%d body=%s", resp.StatusCode, string(data))
	}
	log.Printf("resend accepted to=%s subject=%q", to, subject)
	return nil
}

func WelcomeSubject() string {
	return "Welcome to KazakhExpress"
}

func WelcomeBody(firstName string) string {
	if firstName == "" {
		firstName = "there"
	}
	return fmt.Sprintf("Hi %s,\n\nWelcome to KazakhExpress. Your account is ready, and you can now place orders, pay safely, and review products after checkout.\n\nThanks for joining us.", firstName)
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

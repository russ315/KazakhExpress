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
	"time"
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

func getEmailType(subject string) string {
	switch {
	case subject == WelcomeSubject():
		return "welcome"
	case subject == PaymentReceiptSubject():
		return "receipt"
	case subject == PaymentRefundSubject():
		return "refund"
	case subject == PaymentFailureSubject():
		return "failure"
	default:
		return "unknown"
	}
}

func (s *Service) SendEmail(ctx context.Context, to string, subject string, body string) error {
	emailType := getEmailType(subject)
	start := time.Now()

	if to == "" || subject == "" || body == "" {
		EmailFailuresTotal.WithLabelValues(emailType, "missing_arguments").Inc()
		return fmt.Errorf("email to, subject and body are required")
	}

	if s.config.ResendAPIKey != "" {
		err := s.sendResend(ctx, to, subject, body)
		duration := time.Since(start).Seconds()
		if err != nil {
			EmailFailuresTotal.WithLabelValues(emailType, "resend_error").Inc()
			return err
		}
		EmailsSentTotal.WithLabelValues(emailType).Inc()
		EmailDeliveryDurationSeconds.WithLabelValues(emailType, "resend").Observe(duration)
		return nil
	}

	if s.config.Username == "" || s.config.Password == "" {
		log.Printf("smtp dry-run to=%s subject=%q", to, subject)
		duration := time.Since(start).Seconds()
		EmailsSentTotal.WithLabelValues(emailType).Inc()
		EmailDeliveryDurationSeconds.WithLabelValues(emailType, "dry_run").Observe(duration)
		return nil
	}

	select {
	case <-ctx.Done():
		EmailFailuresTotal.WithLabelValues(emailType, "context_cancelled").Inc()
		return ctx.Err()
	default:
	}

	auth := netsmtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.config.From, to, subject, body)
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	if err := netsmtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(message)); err != nil {
		EmailFailuresTotal.WithLabelValues(emailType, "smtp_error").Inc()
		return fmt.Errorf("send smtp email: %w", err)
	}

	duration := time.Since(start).Seconds()
	EmailsSentTotal.WithLabelValues(emailType).Inc()
	EmailDeliveryDurationSeconds.WithLabelValues(emailType, "smtp").Observe(duration)
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

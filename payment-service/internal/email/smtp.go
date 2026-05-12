package email

import (
	"context"
	"fmt"
	"net/smtp"

	"kazakhexpress/payment-service/internal/payment"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type SMTPSender struct {
	config SMTPConfig
}

func NewSMTPSender(config SMTPConfig) *SMTPSender {
	return &SMTPSender{config: config}
}

func (s *SMTPSender) SendReceipt(ctx context.Context, email payment.ReceiptEmail) error {
	subject := "KazakhExpress payment receipt"
	body := fmt.Sprintf("Payment %s for order %s was paid successfully. Amount: %d KZT.", email.PaymentID, email.OrderID, email.AmountKZT)
	return s.send(ctx, email.To, subject, body)
}

func (s *SMTPSender) SendRefund(ctx context.Context, email payment.RefundEmail) error {
	subject := "KazakhExpress payment refund"
	body := fmt.Sprintf("Payment %s for order %s was refunded. Amount: %d KZT. Reason: %s.", email.PaymentID, email.OrderID, email.AmountKZT, email.Reason)
	return s.send(ctx, email.To, subject, body)
}

func (s *SMTPSender) send(ctx context.Context, to string, subject string, body string) error {
	if s.config.Username == "" || s.config.Password == "" {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", s.config.From, to, subject, body)
	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	if err := smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("send smtp email: %w", err)
	}
	return nil
}

package smtp

import (
	"context"
	"testing"
)

func TestSendEmailSkipsNetworkWhenCredentialsAreMissing(t *testing.T) {
	service := NewService(Config{Host: "smtp.example.com", Port: "587", From: "noreply@example.com"})

	if err := service.SendEmail(context.Background(), "buyer@example.com", "Subject", "Body"); err != nil {
		t.Fatalf("SendEmail() error = %v", err)
	}
}

func TestSendEmailRejectsMissingRequiredFields(t *testing.T) {
	service := NewService(Config{})

	if err := service.SendEmail(context.Background(), "", "Subject", "Body"); err == nil {
		t.Fatal("SendEmail() error = nil, want validation error")
	}
}

func TestPaymentReceiptBodyIncludesPaymentDetails(t *testing.T) {
	body := PaymentReceiptBody("pay-1", "ord-1", 25000)

	if body != "Payment pay-1 for order ord-1 was paid successfully. Amount: 25000 KZT." {
		t.Fatalf("body = %q", body)
	}
}

package provider

import (
	"context"

	"kazakhexpress/payment-service/internal/payment"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Charge(ctx context.Context, pay payment.Payment) (payment.ProviderResult, error) {
	if pay.Method == payment.MethodWallet && pay.AmountKZT > 500000 {
		return payment.ProviderResult{
			Status:                payment.StatusFailed,
			ProviderTransactionID: "mock-" + pay.ID,
			FailureReason:         "wallet limit exceeded",
		}, nil
	}
	return payment.ProviderResult{
		Status:                payment.StatusSucceeded,
		ProviderTransactionID: "mock-" + pay.ID,
	}, nil
}

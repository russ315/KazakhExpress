package payment

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusRefunded  Status = "refunded"
	StatusCancelled Status = "cancelled"

	StatusPaid Status = StatusSucceeded
)

type Method string

const (
	MethodCard   Method = "card"
	MethodKaspi  Method = "kaspi"
	MethodWallet Method = "wallet"
)

type Payment struct {
	ID                    string    `json:"id"`
	OrderID               string    `json:"order_id"`
	CustomerID            string    `json:"customer_id"`
	CustomerEmail         string    `json:"customer_email"`
	AmountKZT             int64     `json:"amount_kzt"`
	Method                Method    `json:"method"`
	Status                Status    `json:"status"`
	ProviderTransactionID string    `json:"provider_transaction_id,omitempty"`
	IdempotencyKey        string    `json:"idempotency_key,omitempty"`
	RefundReason          string    `json:"refund_reason,omitempty"`
	FailureReason         string    `json:"failure_reason,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type CreateInput struct {
	OrderID        string `json:"order_id"`
	CustomerID     string `json:"customer_id"`
	CustomerEmail  string `json:"customer_email"`
	AmountKZT      int64  `json:"amount_kzt"`
	Method         Method `json:"method"`
	IdempotencyKey string `json:"idempotency_key"`
}

type RefundInput struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason"`
}

type CancelInput struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason"`
}

type ConfirmInput struct {
	PaymentID             string `json:"payment_id"`
	ProviderTransactionID string `json:"provider_transaction_id"`
}

type ListFilter struct {
	CustomerID string
}

type ProviderResult struct {
	Status                Status
	ProviderTransactionID string
	FailureReason         string
}

type PaymentEvent struct {
	PaymentID             string    `json:"payment_id"`
	OrderID               string    `json:"order_id"`
	CustomerID            string    `json:"customer_id"`
	AmountKZT             int64     `json:"amount_kzt"`
	Status                Status    `json:"status"`
	Reason                string    `json:"reason,omitempty"`
	ProviderTransactionID string    `json:"provider_transaction_id,omitempty"`
	OccurredAt            time.Time `json:"occurred_at"`
}

type ReceiptEmail struct {
	To        string
	PaymentID string
	OrderID   string
	AmountKZT int64
}

type RefundEmail struct {
	To        string
	PaymentID string
	OrderID   string
	AmountKZT int64
	Reason    string
}

type FailureEmail struct {
	To        string
	PaymentID string
	OrderID   string
	AmountKZT int64
	Reason    string
}

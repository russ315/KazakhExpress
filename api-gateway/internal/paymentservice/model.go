package paymentservice

type Payment struct {
	ID                    string `json:"id"`
	OrderID               string `json:"order_id"`
	CustomerID            string `json:"customer_id"`
	CustomerEmail         string `json:"customer_email"`
	AmountKZT             int64  `json:"amount_kzt"`
	Method                string `json:"method"`
	Status                string `json:"status"`
	ProviderTransactionID string `json:"provider_transaction_id,omitempty"`
	IdempotencyKey        string `json:"idempotency_key,omitempty"`
	RefundReason          string `json:"refund_reason,omitempty"`
	FailureReason         string `json:"failure_reason,omitempty"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

type CreatePaymentRequest struct {
	OrderID        string `json:"order_id"`
	CustomerID     string `json:"customer_id"`
	CustomerEmail  string `json:"customer_email"`
	AmountKZT      int64  `json:"amount_kzt"`
	Method         string `json:"method"`
	IdempotencyKey string `json:"idempotency_key"`
}

type GetPaymentRequest struct {
	PaymentID string `json:"payment_id"`
}

type GetPaymentByOrderIDRequest struct {
	OrderID string `json:"order_id"`
}

type ListPaymentsRequest struct {
	CustomerID string `json:"customer_id"`
}

type ListPaymentsResponse struct {
	Payments []Payment `json:"payments"`
}

type RefundPaymentRequest struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason"`
}

type ConfirmPaymentRequest struct {
	PaymentID             string `json:"payment_id"`
	ProviderTransactionID string `json:"provider_transaction_id"`
}

type CancelPaymentRequest struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason"`
}

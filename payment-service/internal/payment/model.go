package payment

import "time"

type Status string

const (
	StatusPaid     Status = "paid"
	StatusRefunded Status = "refunded"
	StatusFailed   Status = "failed"
)

type Method string

const (
	MethodCard   Method = "card"
	MethodKaspi  Method = "kaspi"
	MethodWallet Method = "wallet"
)

type Payment struct {
	ID            string    `json:"id"`
	OrderID       string    `json:"order_id"`
	CustomerID    string    `json:"customer_id"`
	CustomerEmail string    `json:"customer_email"`
	AmountKZT     int64     `json:"amount_kzt"`
	Method        Method    `json:"method"`
	Status        Status    `json:"status"`
	RefundReason  string    `json:"refund_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateInput struct {
	OrderID       string `json:"order_id"`
	CustomerID    string `json:"customer_id"`
	CustomerEmail string `json:"customer_email"`
	AmountKZT     int64  `json:"amount_kzt"`
	Method        Method `json:"method"`
}

type RefundInput struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason"`
}

type PaymentEvent struct {
	PaymentID  string `json:"payment_id"`
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	AmountKZT  int64  `json:"amount_kzt"`
	Status     Status `json:"status"`
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

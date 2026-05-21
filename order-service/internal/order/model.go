package order

import "time"

type Status string

const (
	StatusCreated       Status = "created"
	StatusPaid          Status = "paid"
	StatusPaymentFailed Status = "payment_failed"
	StatusShipped       Status = "shipped"
	StatusCompleted     Status = "completed"
	StatusCanceled      Status = "canceled"
)

type Item struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	PriceKZT  int64  `json:"price_kzt"`
}

type Order struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Items      []Item    `json:"items"`
	Status     Status    `json:"status"`
	TotalKZT   int64     `json:"total_kzt"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateInput struct {
	CustomerID string `json:"customer_id"`
	Items      []Item `json:"items"`
}

type UpdateStatusInput struct {
	Status Status `json:"status"`
}

type CancelInput struct {
	Reason string `json:"reason"`
}

type StatusHistory struct {
	ID        int64     `json:"id"`
	OrderID   string    `json:"order_id"`
	From      Status    `json:"from_status,omitempty"`
	To        Status    `json:"to_status"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Event struct {
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Status     Status    `json:"status"`
	TotalKZT   int64     `json:"total_kzt"`
	Reason     string    `json:"reason,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

type PaymentEvent struct {
	PaymentID  string    `json:"payment_id"`
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Status     string    `json:"status"`
	Reason     string    `json:"reason,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

type StockReservedEvent struct {
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

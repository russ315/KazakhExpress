package order

import "time"

type Status string

const (
	StatusCreated   Status = "created"
	StatusPaid      Status = "paid"
	StatusShipped   Status = "shipped"
	StatusCompleted Status = "completed"
	StatusCanceled  Status = "canceled"
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

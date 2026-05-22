package review

import "time"

type Review struct {
	ID         string    `json:"id"`
	ProductID  string    `json:"product_id"`
	CustomerID string    `json:"customer_id"`
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Rating struct {
	ProductID string  `json:"product_id"`
	Average   float64 `json:"average_rating"`
	Count     int64   `json:"review_count"`
}

type CreateInput struct {
	ProductID  string `json:"product_id"`
	CustomerID string `json:"customer_id"`
	Rating     int    `json:"rating"`
	Comment    string `json:"comment"`
}

type UpdateInput struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

type ListFilter struct {
	ProductID string
	Limit     int
	Offset    int
}

type Event struct {
	ReviewID   string    `json:"review_id"`
	ProductID  string    `json:"product_id"`
	CustomerID string    `json:"customer_id,omitempty"`
	Rating     int       `json:"rating,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

package review

import "time"

const DefaultPageSize = 20

type Review struct {
	ID           string
	ProductID    string
	UserID       string
	OrderID      string
	Rating       int
	Body         string
	HelpfulCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ProductRating struct {
	ProductID   string
	RatingAvg   float64
	RatingCount int
	UpdatedAt   time.Time
}

type CreateInput struct {
	ProductID string
	UserID    string
	OrderID   string
	Rating    int
	Body      string
}

type UpdateInput struct {
	Rating *int
	Body   *string
}

type ListPage struct {
	Reviews  []Review
	Page     int
	PageSize int
	Total    int
}

type OrderCompletedEvent struct {
	OrderID    string              `json:"order_id"`
	CustomerID string              `json:"customer_id"`
	Items      []OrderCompletedItem `json:"items"`
}

type OrderCompletedItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

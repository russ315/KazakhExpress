package product

import "time"

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceKZT    int64     `json:"price_kzt"`
	Stock       int       `json:"stock"`
	ImageURL    string    `json:"image_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProductImage struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Object    string    `json:"object_name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceKZT    int64  `json:"price_kzt"`
	Stock       int    `json:"stock"`
}

type UpdateStockInput struct {
	Stock int `json:"stock"`
}

type ListFilter struct {
	Limit  int
	Offset int
	Query  string
}

type ImageInput struct {
	ProductID   string
	Filename    string
	ContentType string
	Content     []byte
}

type Event struct {
	ProductID  string    `json:"product_id"`
	Name       string    `json:"name,omitempty"`
	Stock      int       `json:"stock,omitempty"`
	Quantity   int       `json:"quantity,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

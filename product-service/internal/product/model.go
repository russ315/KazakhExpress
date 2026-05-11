package product

import "time"

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceKZT    int64     `json:"price_kzt"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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

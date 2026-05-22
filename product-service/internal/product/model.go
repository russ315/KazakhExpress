package product

import "time"

type Product struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	PriceKZT      int64     `json:"price_kzt"`
	Stock         int       `json:"stock"`
	ReservedStock int       `json:"reserved_stock"`
	Available     int       `json:"available"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceKZT    int64  `json:"price_kzt"`
	Stock       int    `json:"stock"`
}

type UpdateInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	PriceKZT    *int64  `json:"price_kzt,omitempty"`
}

type UpdateStockInput struct {
	Stock int `json:"stock"`
}

type ReserveStockInput struct {
	Quantity      int    `json:"quantity"`
	ReservationID string `json:"reservation_id"`
}

type ReleaseStockInput struct {
	Quantity      int    `json:"quantity"`
	ReservationID string `json:"reservation_id"`
}

func WithAvailability(p Product) Product {
	p.Available = p.Stock - p.ReservedStock
	if p.Available < 0 {
		p.Available = 0
	}
	return p
}

package productservice

type Product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceKZT    int64  `json:"price_kzt"`
	Stock       int    `json:"stock"`
	ImageURL    string `json:"image_url,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProductImage struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	Object    string `json:"object_name"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

type CreateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceKZT    int64  `json:"price_kzt"`
	Stock       int    `json:"stock"`
}

type UpdateStockRequest struct {
	Stock int `json:"stock"`
}

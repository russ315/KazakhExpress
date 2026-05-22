package reviewservice

type Review struct {
	ID         string `json:"id"`
	ProductID  string `json:"product_id"`
	CustomerID string `json:"customer_id"`
	Rating     int    `json:"rating"`
	Comment    string `json:"comment"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type Rating struct {
	ProductID string  `json:"product_id"`
	Average   float64 `json:"average_rating"`
	Count     int64   `json:"review_count"`
}

type CreateReviewRequest struct {
	CustomerID string `json:"customer_id"`
	Rating     int    `json:"rating"`
	Comment    string `json:"comment"`
}

type UpdateReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

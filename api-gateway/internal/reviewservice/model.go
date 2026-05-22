package reviewservice

type Review struct {
	ID           string `json:"id"`
	ProductID    string `json:"product_id"`
	UserID       string `json:"user_id"`
	OrderID      string `json:"order_id"`
	Rating       int    `json:"rating"`
	Body         string `json:"body"`
	HelpfulCount int    `json:"helpful_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ProductRating struct {
	ProductID   string  `json:"product_id"`
	RatingAvg   float64 `json:"rating_avg"`
	RatingCount int     `json:"rating_count"`
}

type CreateReviewRequest struct {
	UserID  string `json:"user_id"`
	OrderID string `json:"order_id"`
	Rating  int    `json:"rating"`
	Body    string `json:"body"`
}

type UpdateReviewRequest struct {
	Rating *int    `json:"rating,omitempty"`
	Body   *string `json:"body,omitempty"`
}

type ListReviewsResponse struct {
	Reviews  []Review `json:"reviews"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Total    int      `json:"total"`
}

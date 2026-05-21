package orderservice

type Status string

type Item struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Quantity  int    `json:"quantity"`
	PriceKZT  int64  `json:"price_kzt"`
}

type Order struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Items      []Item `json:"items"`
	Status     Status `json:"status"`
	TotalKZT   int64  `json:"total_kzt"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type CreateOrderRequest struct {
	CustomerID string `json:"customer_id"`
	Items      []Item `json:"items"`
}

type GetOrderRequest struct {
	OrderID string `json:"order_id"`
}

type ListOrdersRequest struct{}

type ListOrdersResponse struct {
	Orders []Order `json:"orders"`
}

type UpdateOrderStatusRequest struct {
	OrderID string `json:"order_id"`
	Status  Status `json:"status"`
}

type CancelOrderRequest struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

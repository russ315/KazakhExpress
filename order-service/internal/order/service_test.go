package order

import (
	"context"
	"testing"
)

type fakeRepo struct {
	orders map[string]Order
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{orders: make(map[string]Order)}
}

func (r *fakeRepo) Create(ctx context.Context, order Order) (Order, error) {
	r.orders[order.ID] = order
	return order, nil
}

func (r *fakeRepo) List(ctx context.Context) ([]Order, error) {
	orders := make([]Order, 0, len(r.orders))
	for _, order := range r.orders {
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id string) (Order, error) {
	order, ok := r.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	return order, nil
}

func (r *fakeRepo) UpdateStatus(ctx context.Context, id string, from Status, to Status, reason string) (Order, error) {
	order, ok := r.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	order.Status = to
	r.orders[id] = order
	return order, nil
}

type fakePublisher struct {
	created   int
	cancelled int
	completed int
}

func (p *fakePublisher) PublishOrderCreated(ctx context.Context, event Event) error {
	p.created++
	return nil
}

func (p *fakePublisher) PublishOrderCancelled(ctx context.Context, event Event) error {
	p.cancelled++
	return nil
}

func (p *fakePublisher) PublishOrderCompleted(ctx context.Context, event Event) error {
	p.completed++
	return nil
}

type fakeCache struct {
	status map[string]Status
}

func (c *fakeCache) SetStatus(ctx context.Context, orderID string, status Status) error {
	c.status[orderID] = status
	return nil
}

func TestCreatePublishesCreatedAndCachesStatus(t *testing.T) {
	publisher := &fakePublisher{}
	cache := &fakeCache{status: make(map[string]Status)}
	service := NewService(newFakeRepo(), publisher, cache)

	created, err := service.Create(context.Background(), CreateInput{
		CustomerID: "customer-1",
		Items: []Item{{
			ProductID: "product-1",
			Name:      "Shapan",
			Quantity:  2,
			PriceKZT:  1000,
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusCreated {
		t.Fatalf("status = %s, want %s", created.Status, StatusCreated)
	}
	if created.TotalKZT != 2000 {
		t.Fatalf("total = %d, want 2000", created.TotalKZT)
	}
	if publisher.created != 1 {
		t.Fatalf("created events = %d, want 1", publisher.created)
	}
	if cache.status[created.ID] != StatusCreated {
		t.Fatalf("cached status = %s, want %s", cache.status[created.ID], StatusCreated)
	}
}

func TestHandlePaymentEventsUpdateStatus(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, nil, &fakeCache{status: make(map[string]Status)})
	created, err := service.Create(context.Background(), CreateInput{
		CustomerID: "customer-1",
		Items: []Item{{
			ProductID: "product-1",
			Name:      "Shapan",
			Quantity:  1,
			PriceKZT:  1000,
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := service.HandlePaymentSucceeded(context.Background(), PaymentEvent{OrderID: created.ID}); err != nil {
		t.Fatalf("HandlePaymentSucceeded() error = %v", err)
	}
	paid, _ := repo.GetByID(context.Background(), created.ID)
	if paid.Status != StatusPaid {
		t.Fatalf("status = %s, want %s", paid.Status, StatusPaid)
	}

	if err := service.HandlePaymentFailed(context.Background(), PaymentEvent{OrderID: created.ID, Reason: "declined"}); err != nil {
		t.Fatalf("HandlePaymentFailed() error = %v", err)
	}
	failed, _ := repo.GetByID(context.Background(), created.ID)
	if failed.Status != StatusPaymentFailed {
		t.Fatalf("status = %s, want %s", failed.Status, StatusPaymentFailed)
	}
}

func TestCancelPublishesCancelled(t *testing.T) {
	publisher := &fakePublisher{}
	service := NewService(newFakeRepo(), publisher, nil)
	created, err := service.Create(context.Background(), CreateInput{
		CustomerID: "customer-1",
		Items: []Item{{
			ProductID: "product-1",
			Name:      "Shapan",
			Quantity:  1,
			PriceKZT:  1000,
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	cancelled, err := service.Cancel(context.Background(), created.ID, "customer request")
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if cancelled.Status != StatusCanceled {
		t.Fatalf("status = %s, want %s", cancelled.Status, StatusCanceled)
	}
	if publisher.cancelled != 1 {
		t.Fatalf("cancelled events = %d, want 1", publisher.cancelled)
	}
}

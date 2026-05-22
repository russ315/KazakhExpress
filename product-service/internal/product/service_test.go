package product

import (
	"context"
	"testing"
	"time"
)

type fakeCache struct {
	products map[string]Product
}

func newFakeCache() *fakeCache {
	return &fakeCache{products: make(map[string]Product)}
}

func (c *fakeCache) GetProduct(ctx context.Context, id string) (Product, bool, error) {
	p, ok := c.products[id]
	return p, ok, nil
}

func (c *fakeCache) SetProduct(ctx context.Context, p Product, _ time.Duration) error {
	c.products[p.ID] = p
	return nil
}

func (c *fakeCache) InvalidateProduct(ctx context.Context, id string) error {
	delete(c.products, id)
	return nil
}

type fakePublisher struct {
	stockReserved []StockEvent
}

func (p *fakePublisher) PublishProductCreated(context.Context, ProductEvent) error { return nil }
func (p *fakePublisher) PublishProductUpdated(context.Context, ProductEvent) error { return nil }
func (p *fakePublisher) PublishProductDeleted(context.Context, ProductEvent) error { return nil }
func (p *fakePublisher) PublishStockReserved(_ context.Context, e StockEvent) error {
	p.stockReserved = append(p.stockReserved, e)
	return nil
}
func (p *fakePublisher) PublishStockReleased(context.Context, StockEvent) error { return nil }

func TestProductCRUD(t *testing.T) {
	svc := NewService(NewMemoryRepository(), newFakeCache(), &fakePublisher{})
	ctx := context.Background()

	created, err := svc.Create(ctx, CreateInput{Name: "Shapan", PriceKZT: 1000, Stock: 10})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.Update(ctx, created.ID, UpdateInput{Name: strPtr("Updated")})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Updated" {
		t.Fatal("expected updated name")
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestStockReserveAndRelease(t *testing.T) {
	pub := &fakePublisher{}
	svc := NewService(NewMemoryRepository(), newFakeCache(), pub)
	ctx := context.Background()

	created, _ := svc.Create(ctx, CreateInput{Name: "Boots", PriceKZT: 500, Stock: 5})
	reserved, err := svc.ReserveStock(ctx, created.ID, ReserveStockInput{Quantity: 3, ReservationID: "res-1"})
	if err != nil || reserved.Available != 2 {
		t.Fatalf("reserve failed: %v %+v", err, reserved)
	}

	_, err = svc.ReserveStock(ctx, created.ID, ReserveStockInput{Quantity: 5, ReservationID: "res-2"})
	if err != ErrInsufficientStock {
		t.Fatalf("expected insufficient stock")
	}

	released, err := svc.ReleaseStock(ctx, created.ID, ReleaseStockInput{ReservationID: "res-1", Quantity: 3})
	if err != nil || released.Available != 5 {
		t.Fatalf("release failed: %v %+v", err, released)
	}
}

func strPtr(s string) *string { return &s }

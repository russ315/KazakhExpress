package product

import (
	"context"
	"testing"
)

type testStorage struct{}

func (testStorage) Save(ctx context.Context, objectName, contentType string, content []byte) (string, error) {
	return "http://minio/" + objectName, nil
}

type testPublisher struct {
	created  int
	reserved int
	released int
}

func (p *testPublisher) PublishProductCreated(ctx context.Context, event Event) error {
	p.created++
	return nil
}
func (p *testPublisher) PublishStockUpdated(ctx context.Context, event Event) error { return nil }
func (p *testPublisher) PublishStockReserved(ctx context.Context, event Event) error {
	p.reserved++
	return nil
}
func (p *testPublisher) PublishStockReleased(ctx context.Context, event Event) error {
	p.released++
	return nil
}

func TestServiceCreateReserveReleaseAndImage(t *testing.T) {
	ctx := context.Background()
	pub := &testPublisher{}
	svc := NewService(NewMemoryRepository(), testStorage{}, pub)

	created, err := svc.Create(ctx, CreateInput{Name: "Shapan", PriceKZT: 12000, Stock: 5})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if pub.created != 1 {
		t.Fatalf("created events = %d, want 1", pub.created)
	}

	reserved, err := svc.ReserveStock(ctx, created.ID, 2)
	if err != nil {
		t.Fatalf("ReserveStock() error = %v", err)
	}
	if reserved.Stock != 3 {
		t.Fatalf("stock = %d, want 3", reserved.Stock)
	}

	released, err := svc.ReleaseStock(ctx, created.ID, 1)
	if err != nil {
		t.Fatalf("ReleaseStock() error = %v", err)
	}
	if released.Stock != 4 {
		t.Fatalf("stock = %d, want 4", released.Stock)
	}

	image, err := svc.AddImage(ctx, ImageInput{ProductID: created.ID, Filename: "a.png", ContentType: "image/png", Content: []byte("png")})
	if err != nil {
		t.Fatalf("AddImage() error = %v", err)
	}
	if image.URL == "" {
		t.Fatal("image url is empty")
	}
}

func TestServiceRejectsInvalidProduct(t *testing.T) {
	svc := NewService(NewMemoryRepository(), nil, nil)
	if _, err := svc.Create(context.Background(), CreateInput{Name: "", PriceKZT: -1}); err != ErrInvalidInput {
		t.Fatalf("Create() error = %v, want %v", err, ErrInvalidInput)
	}
}

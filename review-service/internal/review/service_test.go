package review

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type fakeCache struct {
	ratings map[string]ProductRating
	pages   map[string]ListPage
}

func newFakeCache() *fakeCache {
	return &fakeCache{ratings: make(map[string]ProductRating), pages: make(map[string]ListPage)}
}

func (c *fakeCache) GetRating(ctx context.Context, productID string) (ProductRating, bool, error) {
	r, ok := c.ratings[productID]
	return r, ok, nil
}
func (c *fakeCache) SetRating(ctx context.Context, rating ProductRating, _ time.Duration) error {
	c.ratings[rating.ProductID] = rating
	return nil
}
func (c *fakeCache) InvalidateRating(ctx context.Context, productID string) error {
	delete(c.ratings, productID)
	return nil
}
func (c *fakeCache) GetReviewsPage(ctx context.Context, productID string, page int) (ListPage, bool, error) {
	key := fmt.Sprintf("%s:%d", productID, page)
	p, ok := c.pages[key]
	return p, ok, nil
}
func (c *fakeCache) SetReviewsPage(ctx context.Context, productID string, page ListPage, _ time.Duration) error {
	c.pages[productID] = page
	return nil
}
func (c *fakeCache) InvalidateReviews(ctx context.Context, productID string) error {
	delete(c.pages, productID)
	return nil
}

type fakePublisher struct {
	created []ReviewEvent
	rating  []RatingEvent
}

func (p *fakePublisher) PublishReviewCreated(_ context.Context, e ReviewEvent) error {
	p.created = append(p.created, e)
	return nil
}
func (p *fakePublisher) PublishReviewUpdated(context.Context, ReviewEvent) error  { return nil }
func (p *fakePublisher) PublishReviewDeleted(context.Context, ReviewEvent) error  { return nil }
func (p *fakePublisher) PublishProductRatingUpdated(_ context.Context, e RatingEvent) error {
	p.rating = append(p.rating, e)
	return nil
}

func TestReviewCRUDAndRatingWithEligibility(t *testing.T) {
	repo := NewMemoryRepository()
	pub := &fakePublisher{}
	svc := NewService(repo, newFakeCache(), pub)
	ctx := context.Background()

	_ = repo.GrantEligibility(ctx, "u1", "prd-1", "ord-1", time.Now().UTC())
	_ = repo.GrantEligibility(ctx, "u2", "prd-1", "ord-2", time.Now().UTC())

	_, err := svc.Create(ctx, CreateInput{ProductID: "prd-1", UserID: "u1", OrderID: "ord-1", Rating: 4, Body: "good"})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}

	_, err = svc.Create(ctx, CreateInput{ProductID: "prd-1", UserID: "u2", OrderID: "ord-2", Rating: 2})
	if err != nil {
		t.Fatalf("create second review: %v", err)
	}

	rating, err := svc.GetProductRating(ctx, "prd-1")
	if err != nil || rating.RatingCount != 2 || rating.RatingAvg != 3 {
		t.Fatalf("unexpected rating: %+v err=%v", rating, err)
	}

	if _, err := svc.Create(ctx, CreateInput{ProductID: "prd-1", UserID: "u1", OrderID: "ord-1", Rating: 3}); err != ErrDuplicateReview {
		t.Fatalf("expected duplicate, got %v", err)
	}

	if _, err := svc.Create(ctx, CreateInput{ProductID: "prd-1", UserID: "u3", OrderID: "ord-3", Rating: 5}); err != ErrNotEligible {
		t.Fatalf("expected not eligible, got %v", err)
	}

	if len(pub.rating) < 2 {
		t.Fatal("expected product.rating.updated events")
	}
}

func TestHandleOrderCompleted(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewService(repo, newFakeCache(), &fakePublisher{})
	ctx := context.Background()

	err := svc.HandleOrderCompleted(ctx, OrderCompletedEvent{
		OrderID:    "ord-99",
		CustomerID: "u9",
		Items:      []OrderCompletedItem{{ProductID: "prd-9", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("handle order completed: %v", err)
	}

	ok, err := repo.IsEligible(ctx, "u9", "prd-9", "ord-99")
	if err != nil || !ok {
		t.Fatalf("expected eligibility, ok=%v err=%v", ok, err)
	}
}

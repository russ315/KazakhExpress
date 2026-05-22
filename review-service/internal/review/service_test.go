package review

import (
	"context"
	"testing"
)

type memoryRepo struct {
	reviews map[string]Review
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{reviews: map[string]Review{}}
}

func (r *memoryRepo) Create(ctx context.Context, review Review) (Review, error) {
	r.reviews[review.ID] = review
	return review, nil
}
func (r *memoryRepo) GetByID(ctx context.Context, id string) (Review, error) {
	review, ok := r.reviews[id]
	if !ok {
		return Review{}, ErrNotFound
	}
	return review, nil
}
func (r *memoryRepo) ListByProduct(ctx context.Context, filter ListFilter) ([]Review, error) {
	var out []Review
	for _, review := range r.reviews {
		if review.ProductID == filter.ProductID {
			out = append(out, review)
		}
	}
	return out, nil
}
func (r *memoryRepo) Update(ctx context.Context, review Review) (Review, error) {
	r.reviews[review.ID] = review
	return review, nil
}
func (r *memoryRepo) Delete(ctx context.Context, id string) error {
	delete(r.reviews, id)
	return nil
}
func (r *memoryRepo) Rating(ctx context.Context, productID string) (Rating, error) {
	var sum, count int
	for _, review := range r.reviews {
		if review.ProductID == productID {
			sum += review.Rating
			count++
		}
	}
	if count == 0 {
		return Rating{ProductID: productID}, nil
	}
	return Rating{ProductID: productID, Average: float64(sum) / float64(count), Count: int64(count)}, nil
}

func TestServiceCreateUpdateRatingDelete(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMemoryRepo(), nil, nil)

	created, err := svc.Create(ctx, CreateInput{ProductID: "prd-1", CustomerID: "usr-1", Rating: 5, Comment: "good"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	updated, err := svc.Update(ctx, created.ID, UpdateInput{Rating: 4, Comment: "ok"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Rating != 4 {
		t.Fatalf("rating = %d, want 4", updated.Rating)
	}
	rating, err := svc.Rating(ctx, "prd-1")
	if err != nil {
		t.Fatalf("Rating() error = %v", err)
	}
	if rating.Average != 4 || rating.Count != 1 {
		t.Fatalf("rating = %+v, want avg 4 count 1", rating)
	}
	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestServiceRejectsInvalidRating(t *testing.T) {
	svc := NewService(newMemoryRepo(), nil, nil)
	if _, err := svc.Create(context.Background(), CreateInput{ProductID: "p", CustomerID: "u", Rating: 6}); err != ErrInvalidInput {
		t.Fatalf("Create() error = %v, want %v", err, ErrInvalidInput)
	}
}

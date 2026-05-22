//go:build integration

package review

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRepositoryReviewLifecycle(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is required for integration tests")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(db.Close)

	repo := NewPostgresRepository(db)
	now := time.Now().UTC()
	id := "it-review-" + now.Format("150405.000000000")
	productID := "it-product-" + id
	r := Review{ID: id, ProductID: productID, CustomerID: "it-user", Rating: 4, Comment: "good", CreatedAt: now, UpdatedAt: now}
	if _, err := repo.Create(ctx, r); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	r.Rating = 5
	r.Comment = "great"
	r.UpdatedAt = time.Now().UTC()
	if _, err := repo.Update(ctx, r); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	rating, err := repo.Rating(ctx, productID)
	if err != nil {
		t.Fatalf("Rating() error = %v", err)
	}
	if rating.Count != 1 || rating.Average != 5 {
		t.Fatalf("rating = %+v, want count 1 average 5", rating)
	}
	if err := repo.Delete(ctx, id); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

//go:build integration

package product

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRepositoryProductLifecycle(t *testing.T) {
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
	id := "it-product-" + now.Format("150405.000000000")
	p := Product{ID: id, Name: "Integration Backpack", Description: "Postgres integration", PriceKZT: 15000, Stock: 9, CreatedAt: now, UpdatedAt: now}
	if _, err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	p.Stock = 7
	p.UpdatedAt = time.Now().UTC()
	if _, err := repo.Update(ctx, p); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	image := ProductImage{ID: "it-image-" + id, ProductID: id, Object: "it.jpg", URL: "http://minio/it.jpg", CreatedAt: time.Now().UTC()}
	if _, err := repo.AddImage(ctx, image); err != nil {
		t.Fatalf("AddImage() error = %v", err)
	}
	found, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if found.Stock != 7 || found.ImageURL == "" {
		t.Fatalf("found = %+v, want stock 7 and image url", found)
	}
}

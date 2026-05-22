//go:build integration

package order

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRepositoryOrderLifecycle(t *testing.T) {
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
	id := "it-order-" + now.Format("150405.000000000")
	order := Order{
		ID: id, CustomerID: "it-user-" + id, Status: StatusCreated, TotalKZT: 42000,
		CreatedAt: now, UpdatedAt: now,
		Items: []Item{{ProductID: "it-product-" + id, Name: "Integration product", Quantity: 2, PriceKZT: 21000}},
	}

	if _, err := repo.Create(ctx, order); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	found, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if len(found.Items) != 1 || found.TotalKZT != 42000 {
		t.Fatalf("found = %+v, want one item and total 42000", found)
	}
	paid, err := repo.UpdateStatus(ctx, id, StatusCreated, StatusPaid, "integration payment")
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if paid.Status != StatusPaid {
		t.Fatalf("status = %s, want %s", paid.Status, StatusPaid)
	}
}

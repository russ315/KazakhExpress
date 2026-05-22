//go:build integration

package payment

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresRepositoryPaymentLifecycle(t *testing.T) {
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
	id := "it-pay-" + time.Now().Format("150405.000000000")
	pay := Payment{
		ID: id, OrderID: "it-order-" + id, CustomerID: "it-user-" + id,
		CustomerEmail: "integration@example.com", AmountKZT: 12000, Method: MethodCard,
		Status: StatusPending, IdempotencyKey: "idem-" + id, CreatedAt: now, UpdatedAt: now,
	}

	created, err := repo.Create(ctx, pay)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != id {
		t.Fatalf("Create() ID = %q, want %q", created.ID, id)
	}

	created.Status = StatusSucceeded
	created.ProviderTransactionID = "provider-" + id
	created.UpdatedAt = time.Now().UTC()
	if _, err := repo.Update(ctx, created); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	found, err := repo.GetByOrderID(ctx, created.OrderID)
	if err != nil {
		t.Fatalf("GetByOrderID() error = %v", err)
	}
	if found.Status != StatusSucceeded {
		t.Fatalf("status = %s, want %s", found.Status, StatusSucceeded)
	}

	if err := repo.AppendEvent(ctx, PaymentEvent{
		PaymentID: found.ID, OrderID: found.OrderID, CustomerID: found.CustomerID,
		AmountKZT: found.AmountKZT, Status: StatusRefunded, Reason: "integration",
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}
}

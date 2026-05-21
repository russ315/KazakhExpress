package order

import (
	"context"
	"errors"
	"testing"
)

func validCreateInput() CreateInput {
	return CreateInput{
		CustomerID: "customer-1",
		Items: []Item{{
			ProductID: "product-1",
			Name:      "Shapan",
			Quantity:  2,
			PriceKZT:  1000,
		}},
	}
}

func TestCreatePublishesCreatedAndCachesStatus(t *testing.T) {
	publisher := &mockPublisher{}
	cache := newMockCache()
	service := NewService(newFakeRepo(), publisher, cache)

	created, err := service.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusCreated {
		t.Fatalf("status = %s, want %s", created.Status, StatusCreated)
	}
	if created.TotalKZT != 2000 {
		t.Fatalf("total = %d, want 2000", created.TotalKZT)
	}
	if len(publisher.createdEvents) != 1 {
		t.Fatalf("created events = %d, want 1", len(publisher.createdEvents))
	}
	if publisher.createdEvents[0].OrderID != created.ID {
		t.Fatalf("event order_id = %s, want %s", publisher.createdEvents[0].OrderID, created.ID)
	}
	if cache.status[created.ID] != StatusCreated {
		t.Fatalf("cached status = %s, want %s", cache.status[created.ID], StatusCreated)
	}
}

func TestCreateValidationErrors(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)

	tests := []struct {
		name  string
		input CreateInput
	}{
		{name: "missing customer", input: CreateInput{Items: validCreateInput().Items}},
		{name: "missing items", input: CreateInput{CustomerID: "customer-1"}},
		{name: "missing product id", input: CreateInput{
			CustomerID: "customer-1",
			Items:      []Item{{Name: "x", Quantity: 1, PriceKZT: 1}},
		}},
		{name: "invalid quantity", input: CreateInput{
			CustomerID: "customer-1",
			Items:      []Item{{ProductID: "p1", Name: "x", Quantity: 0, PriceKZT: 1}},
		}},
		{name: "negative price", input: CreateInput{
			CustomerID: "customer-1",
			Items:      []Item{{ProductID: "p1", Name: "x", Quantity: 1, PriceKZT: -1}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.input)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Create() error = %v, want %v", err, ErrInvalidInput)
			}
		})
	}
}

func TestCreateRepoError(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = errMock
	service := NewService(repo, nil, nil)

	_, err := service.Create(context.Background(), validCreateInput())
	if !errors.Is(err, errMock) {
		t.Fatalf("Create() error = %v, want %v", err, errMock)
	}
}

func TestCreateStillSucceedsWhenPublisherFails(t *testing.T) {
	publisher := &mockPublisher{createdErr: errMock}
	service := NewService(newFakeRepo(), publisher, nil)

	created, err := service.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected order to be created")
	}
}

func TestListAndGetByID(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)
	created, err := service.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	orders, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(orders))
	}

	found, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("order id = %s, want %s", found.ID, created.ID)
	}
}

func TestGetByIDValidationAndNotFound(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)

	_, err := service.GetByID(context.Background(), "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("GetByID() error = %v, want %v", err, ErrInvalidInput)
	}

	_, err = service.GetByID(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want %v", err, ErrNotFound)
	}
}

func TestUpdateStatusToCompletedPublishesEvent(t *testing.T) {
	publisher := &mockPublisher{}
	cache := newMockCache()
	service := NewService(newFakeRepo(), publisher, cache)
	created, err := service.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := service.UpdateStatus(context.Background(), created.ID, StatusCompleted)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if updated.Status != StatusCompleted {
		t.Fatalf("status = %s, want %s", updated.Status, StatusCompleted)
	}
	if len(publisher.completedEvents) != 1 {
		t.Fatalf("completed events = %d, want 1", len(publisher.completedEvents))
	}
	if cache.status[created.ID] != StatusCompleted {
		t.Fatalf("cached status = %s, want %s", cache.status[created.ID], StatusCompleted)
	}
}

func TestUpdateStatusValidation(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)

	_, err := service.UpdateStatus(context.Background(), "", StatusPaid)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("UpdateStatus() error = %v, want %v", err, ErrInvalidInput)
	}

	_, err = service.UpdateStatus(context.Background(), "ord-1", Status("unknown"))
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("UpdateStatus() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestHandlePaymentEventsUpdateStatus(t *testing.T) {
	repo := newFakeRepo()
	cache := newMockCache()
	service := NewService(repo, nil, cache)
	created, err := service.Create(context.Background(), validCreateInput())
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
	if cache.status[created.ID] != StatusPaymentFailed {
		t.Fatalf("cached status = %s, want %s", cache.status[created.ID], StatusPaymentFailed)
	}
}

func TestHandlePaymentEventsValidation(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)

	err := service.HandlePaymentSucceeded(context.Background(), PaymentEvent{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("HandlePaymentSucceeded() error = %v, want %v", err, ErrInvalidInput)
	}

	err = service.HandlePaymentFailed(context.Background(), PaymentEvent{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("HandlePaymentFailed() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestHandlePaymentSucceededNotFound(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)
	err := service.HandlePaymentSucceeded(context.Background(), PaymentEvent{OrderID: "missing"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("HandlePaymentSucceeded() error = %v, want %v", err, ErrNotFound)
	}
}

func TestHandleStockReserved(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)
	created, err := service.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := service.HandleStockReserved(context.Background(), StockReservedEvent{OrderID: created.ID}); err != nil {
		t.Fatalf("HandleStockReserved() error = %v", err)
	}

	err = service.HandleStockReserved(context.Background(), StockReservedEvent{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("HandleStockReserved() error = %v, want %v", err, ErrInvalidInput)
	}

	err = service.HandleStockReserved(context.Background(), StockReservedEvent{OrderID: "missing"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("HandleStockReserved() error = %v, want %v", err, ErrNotFound)
	}
}

func TestCancelPublishesCancelled(t *testing.T) {
	publisher := &mockPublisher{}
	service := NewService(newFakeRepo(), publisher, nil)
	created, err := service.Create(context.Background(), validCreateInput())
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
	if len(publisher.cancelledEvents) != 1 {
		t.Fatalf("cancelled events = %d, want 1", len(publisher.cancelledEvents))
	}
	if publisher.cancelledEvents[0].Reason != "customer request" {
		t.Fatalf("reason = %q, want %q", publisher.cancelledEvents[0].Reason, "customer request")
	}
}

func TestCancelAlreadyTerminalStatus(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)
	created, err := service.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := service.UpdateStatus(context.Background(), created.ID, StatusCompleted); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	_, err = service.Cancel(context.Background(), created.ID, "too late")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Cancel() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestCancelEmptyID(t *testing.T) {
	service := NewService(newFakeRepo(), nil, nil)
	_, err := service.Cancel(context.Background(), "", "reason")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Cancel() error = %v, want %v", err, ErrInvalidInput)
	}
}

func TestCreateStillSucceedsWhenCacheFails(t *testing.T) {
	cache := newMockCache()
	cache.setErr = errMock
	service := NewService(newFakeRepo(), nil, cache)

	created, err := service.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusCreated {
		t.Fatalf("status = %s, want %s", created.Status, StatusCreated)
	}
}

func TestEventFromOrder(t *testing.T) {
	order := Order{
		ID:         "ord-1",
		CustomerID: "customer-1",
		Status:     StatusPaid,
		TotalKZT:   5000,
	}
	event := eventFromOrder(order, "test reason")
	if event.OrderID != order.ID || event.CustomerID != order.CustomerID {
		t.Fatalf("unexpected event payload: %+v", event)
	}
	if event.Status != StatusPaid || event.TotalKZT != 5000 || event.Reason != "test reason" {
		t.Fatalf("unexpected event fields: %+v", event)
	}
	if event.OccurredAt.IsZero() {
		t.Fatal("expected occurred_at to be set")
	}
}

package order

import (
	"context"
	"errors"
)

var errMock = errors.New("mock error")

// fakeRepo is an in-memory Repository for unit tests.
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

// mockRepo injects errors into Repository calls.
type mockRepo struct {
	*fakeRepo
	createErr      error
	listErr        error
	getErr         error
	updateErr      error
	updateCalls    int
	lastUpdateFrom Status
	lastUpdateTo   Status
	lastReason     string
}

func newMockRepo() *mockRepo {
	return &mockRepo{fakeRepo: newFakeRepo()}
}

func (r *mockRepo) Create(ctx context.Context, order Order) (Order, error) {
	if r.createErr != nil {
		return Order{}, r.createErr
	}
	return r.fakeRepo.Create(ctx, order)
}

func (r *mockRepo) List(ctx context.Context) ([]Order, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.fakeRepo.List(ctx)
}

func (r *mockRepo) GetByID(ctx context.Context, id string) (Order, error) {
	if r.getErr != nil {
		return Order{}, r.getErr
	}
	return r.fakeRepo.GetByID(ctx, id)
}

func (r *mockRepo) UpdateStatus(ctx context.Context, id string, from Status, to Status, reason string) (Order, error) {
	r.updateCalls++
	r.lastUpdateFrom = from
	r.lastUpdateTo = to
	r.lastReason = reason
	if r.updateErr != nil {
		return Order{}, r.updateErr
	}
	return r.fakeRepo.UpdateStatus(ctx, id, from, to, reason)
}

// mockPublisher records published events and can fail on demand.
type mockPublisher struct {
	createdEvents   []Event
	cancelledEvents []Event
	completedEvents []Event
	createdErr      error
	cancelledErr    error
	completedErr    error
}

func (p *mockPublisher) PublishOrderCreated(ctx context.Context, event Event) error {
	p.createdEvents = append(p.createdEvents, event)
	return p.createdErr
}

func (p *mockPublisher) PublishOrderCancelled(ctx context.Context, event Event) error {
	p.cancelledEvents = append(p.cancelledEvents, event)
	return p.cancelledErr
}

func (p *mockPublisher) PublishOrderCompleted(ctx context.Context, event Event) error {
	p.completedEvents = append(p.completedEvents, event)
	return p.completedErr
}

// mockCache records cache writes and can fail on demand.
type mockCache struct {
	status   map[string]Status
	setCalls int
	setErr   error
}

func newMockCache() *mockCache {
	return &mockCache{status: make(map[string]Status)}
}

func (c *mockCache) SetStatus(ctx context.Context, orderID string, status Status) error {
	c.setCalls++
	if c.setErr != nil {
		return c.setErr
	}
	c.status[orderID] = status
	return nil
}

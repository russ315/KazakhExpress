package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("payment not found")

type Repository interface {
	Create(ctx context.Context, payment Payment) (Payment, error)
	GetByID(ctx context.Context, id string) (Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (Payment, error)
	List(ctx context.Context, filter ListFilter) ([]Payment, error)
	Update(ctx context.Context, payment Payment) (Payment, error)
	AppendEvent(ctx context.Context, event PaymentEvent) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, p Payment) (Payment, error) {
	const query = `
		insert into payments (
			id, order_id, customer_id, customer_email, amount_kzt, method, status,
			provider_transaction_id, idempotency_key, refund_reason, failure_reason, created_at, updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.Exec(ctx, query, p.ID, p.OrderID, p.CustomerID, p.CustomerEmail, p.AmountKZT, p.Method, p.Status, p.ProviderTransactionID, p.IdempotencyKey, p.RefundReason, p.FailureReason, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return Payment{}, fmt.Errorf("create payment: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Payment, error) {
	const query = `
		select id, order_id, customer_id, customer_email, amount_kzt, method, status,
		       provider_transaction_id, idempotency_key, refund_reason, failure_reason, created_at, updated_at
		from payments
		where id = $1`

	p, err := scanPayment(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	if err != nil {
		return Payment{}, fmt.Errorf("get payment: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) GetByOrderID(ctx context.Context, orderID string) (Payment, error) {
	const query = `
		select id, order_id, customer_id, customer_email, amount_kzt, method, status,
		       provider_transaction_id, idempotency_key, refund_reason, failure_reason, created_at, updated_at
		from payments
		where order_id = $1
		order by created_at desc
		limit 1`

	p, err := scanPayment(r.db.QueryRow(ctx, query, orderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	if err != nil {
		return Payment{}, fmt.Errorf("get payment by order id: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]Payment, error) {
	query := `
		select id, order_id, customer_id, customer_email, amount_kzt, method, status,
		       provider_transaction_id, idempotency_key, refund_reason, failure_reason, created_at, updated_at
		from payments`
	args := []any{}
	if filter.CustomerID != "" {
		query += ` where customer_id = $1`
		args = append(args, filter.CustomerID)
	}
	query += `
		order by created_at desc`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	payments := make([]Payment, 0)
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payments: %w", err)
	}
	return payments, nil
}

func (r *PostgresRepository) Update(ctx context.Context, p Payment) (Payment, error) {
	const query = `
		update payments
		set status = $2,
		    provider_transaction_id = $3,
		    refund_reason = $4,
		    failure_reason = $5,
		    updated_at = $6
		where id = $1`

	tag, err := r.db.Exec(ctx, query, p.ID, p.Status, p.ProviderTransactionID, p.RefundReason, p.FailureReason, p.UpdatedAt)
	if err != nil {
		return Payment{}, fmt.Errorf("update payment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Payment{}, ErrNotFound
	}
	return p, nil
}

func (r *PostgresRepository) AppendEvent(ctx context.Context, event PaymentEvent) error {
	const query = `
		insert into payment_events (
			payment_id, order_id, customer_id, amount_kzt, status, reason, provider_transaction_id, occurred_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Exec(ctx, query, event.PaymentID, event.OrderID, event.CustomerID, event.AmountKZT, event.Status, event.Reason, event.ProviderTransactionID, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("append payment event: %w", err)
	}
	if event.Status == StatusRefunded {
		const refundQuery = `
			insert into refunds (payment_id, reason, amount_kzt, created_at)
			values ($1, $2, $3, $4)`
		if _, err := r.db.Exec(ctx, refundQuery, event.PaymentID, event.Reason, event.AmountKZT, event.OccurredAt); err != nil {
			return fmt.Errorf("append refund: %w", err)
		}
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPayment(row rowScanner) (Payment, error) {
	var p Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.CustomerID, &p.CustomerEmail, &p.AmountKZT, &p.Method, &p.Status, &p.ProviderTransactionID, &p.IdempotencyKey, &p.RefundReason, &p.FailureReason, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

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
	List(ctx context.Context) ([]Payment, error)
	Update(ctx context.Context, payment Payment) (Payment, error)
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
			id, order_id, customer_id, customer_email, amount_kzt, method, status, refund_reason, created_at, updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(ctx, query, p.ID, p.OrderID, p.CustomerID, p.CustomerEmail, p.AmountKZT, p.Method, p.Status, p.RefundReason, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return Payment{}, fmt.Errorf("create payment: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Payment, error) {
	const query = `
		select id, order_id, customer_id, customer_email, amount_kzt, method, status, refund_reason, created_at, updated_at
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

func (r *PostgresRepository) List(ctx context.Context) ([]Payment, error) {
	const query = `
		select id, order_id, customer_id, customer_email, amount_kzt, method, status, refund_reason, created_at, updated_at
		from payments
		order by created_at desc`

	rows, err := r.db.Query(ctx, query)
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
		set status = $2, refund_reason = $3, updated_at = $4
		where id = $1`

	tag, err := r.db.Exec(ctx, query, p.ID, p.Status, p.RefundReason, p.UpdatedAt)
	if err != nil {
		return Payment{}, fmt.Errorf("update payment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Payment{}, ErrNotFound
	}
	return p, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPayment(row rowScanner) (Payment, error) {
	var p Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.CustomerID, &p.CustomerEmail, &p.AmountKZT, &p.Method, &p.Status, &p.RefundReason, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

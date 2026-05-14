package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("order not found")

type Repository interface {
	Create(ctx context.Context, order Order) (Order, error)
	List(ctx context.Context) ([]Order, error)
	GetByID(ctx context.Context, id string) (Order, error)
	Update(ctx context.Context, order Order) (Order, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, order Order) (Order, error) {
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return Order{}, fmt.Errorf("marshal order items: %w", err)
	}

	const query = `
		insert into orders (
			id, customer_id, items, status, total_kzt, created_at, updated_at
		) values ($1, $2, $3, $4, $5, $6, $7)`

	_, err = r.db.Exec(ctx, query, order.ID, order.CustomerID, itemsJSON, order.Status, order.TotalKZT, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}
	return order, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]Order, error) {
	const query = `
		select id, customer_id, items, status, total_kzt, created_at, updated_at
		from orders
		order by created_at desc`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Order, error) {
	const query = `
		select id, customer_id, items, status, total_kzt, created_at, updated_at
		from orders
		where id = $1`

	order, err := scanOrder(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("get order: %w", err)
	}

	return order, nil
}

func (r *PostgresRepository) Update(ctx context.Context, order Order) (Order, error) {
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return Order{}, fmt.Errorf("marshal order items: %w", err)
	}

	const query = `
		update orders
		set customer_id = $2,
			items = $3,
			status = $4,
			total_kzt = $5,
			updated_at = $6
		where id = $1`

	tag, err := r.db.Exec(ctx, query, order.ID, order.CustomerID, itemsJSON, order.Status, order.TotalKZT, order.UpdatedAt)
	if err != nil {
		return Order{}, fmt.Errorf("update order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Order{}, ErrNotFound
	}

	return order, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrder(row rowScanner) (Order, error) {
	var order Order
	var itemsJSON []byte

	err := row.Scan(&order.ID, &order.CustomerID, &itemsJSON, &order.Status, &order.TotalKZT, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return Order{}, err
	}
	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return Order{}, fmt.Errorf("unmarshal order items: %w", err)
	}

	return order, nil
}

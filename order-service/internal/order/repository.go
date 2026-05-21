package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("order not found")

type Repository interface {
	Create(ctx context.Context, order Order) (Order, error)
	List(ctx context.Context) ([]Order, error)
	GetByID(ctx context.Context, id string) (Order, error)
	UpdateStatus(ctx context.Context, id string, from Status, to Status, reason string) (Order, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, order Order) (Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin create order: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		insert into orders (id, customer_id, status, total_kzt, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6)`,
		order.ID, order.CustomerID, order.Status, order.TotalKZT, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return Order{}, fmt.Errorf("create order: %w", err)
	}

	for _, item := range order.Items {
		_, err = tx.Exec(ctx, `
			insert into order_items (order_id, product_id, name, quantity, price_kzt)
			values ($1, $2, $3, $4, $5)`,
			order.ID, item.ProductID, item.Name, item.Quantity, item.PriceKZT)
		if err != nil {
			return Order{}, fmt.Errorf("create order item: %w", err)
		}
	}

	if err := insertStatusHistory(ctx, tx, order.ID, "", order.Status, "order created", order.CreatedAt); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit create order: %w", err)
	}
	return order, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]Order, error) {
	rows, err := r.db.Query(ctx, `
		select id, customer_id, status, total_kzt, created_at, updated_at
		from orders
		order by created_at desc`)
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
		order.Items, err = r.listItems(ctx, order.ID)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}
	return orders, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Order, error) {
	order, err := scanOrder(r.db.QueryRow(ctx, `
		select id, customer_id, status, total_kzt, created_at, updated_at
		from orders
		where id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("get order: %w", err)
	}
	order.Items, err = r.listItems(ctx, order.ID)
	if err != nil {
		return Order{}, err
	}
	return order, nil
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id string, from Status, to Status, reason string) (Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Order{}, fmt.Errorf("begin update order status: %w", err)
	}
	defer tx.Rollback(ctx)

	updated, err := scanOrder(tx.QueryRow(ctx, `
		update orders
		set status = $2,
			updated_at = now()
		where id = $1
		returning id, customer_id, status, total_kzt, created_at, updated_at`, id, to))
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("update order status: %w", err)
	}
	if err := insertStatusHistory(ctx, tx, id, from, to, reason, updated.UpdatedAt); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, fmt.Errorf("commit update order status: %w", err)
	}
	updated.Items, err = r.listItems(ctx, updated.ID)
	if err != nil {
		return Order{}, err
	}
	return updated, nil
}

func (r *PostgresRepository) listItems(ctx context.Context, orderID string) ([]Item, error) {
	rows, err := r.db.Query(ctx, `
		select product_id, name, quantity, price_kzt
		from order_items
		where order_id = $1
		order by id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ProductID, &item.Name, &item.Quantity, &item.PriceKZT); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}
	return items, nil
}

type statusHistoryExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func insertStatusHistory(ctx context.Context, tx statusHistoryExecutor, orderID string, from Status, to Status, reason string, createdAt time.Time) error {
	_, err := tx.Exec(ctx, `
		insert into order_status_history (order_id, from_status, to_status, reason, created_at)
		values ($1, nullif($2, ''), $3, $4, $5)`,
		orderID, from, to, reason, createdAt)
	if err != nil {
		return fmt.Errorf("insert order status history: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrder(row rowScanner) (Order, error) {
	var order Order
	err := row.Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalKZT, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return Order{}, err
	}
	return order, nil
}

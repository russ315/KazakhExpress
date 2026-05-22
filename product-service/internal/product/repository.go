package product

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("product not found")

type Repository interface {
	Create(ctx context.Context, p Product) (Product, error)
	List(ctx context.Context, filter ListFilter) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	Update(ctx context.Context, p Product) (Product, error)
	AddImage(ctx context.Context, image ProductImage) (ProductImage, error)
}

type MemoryRepository struct {
	mu       sync.RWMutex
	products map[string]Product
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{products: make(map[string]Product)}
}

func (r *MemoryRepository) Create(ctx context.Context, p Product) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[p.ID] = p
	return p, nil
}

func (r *MemoryRepository) List(ctx context.Context, filter ListFilter) ([]Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Product, 0, len(r.products))
	for _, p := range r.products {
		out = append(out, p)
	}
	return out, nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id string) (Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[id]
	if !ok {
		return Product{}, ErrNotFound
	}
	return p, nil
}

func (r *MemoryRepository) Update(ctx context.Context, p Product) (Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.products[p.ID]; !ok {
		return Product{}, ErrNotFound
	}
	r.products[p.ID] = p
	return p, nil
}

func (r *MemoryRepository) AddImage(ctx context.Context, image ProductImage) (ProductImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[image.ProductID]
	if !ok {
		return ProductImage{}, ErrNotFound
	}
	p.ImageURL = image.URL
	p.UpdatedAt = time.Now().UTC()
	r.products[p.ID] = p
	return image, nil
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, p Product) (Product, error) {
	const query = `
		INSERT INTO products (id, name, description, price_kzt, stock, image_url, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	if _, err := r.db.Exec(ctx, query, p.ID, p.Name, p.Description, p.PriceKZT, p.Stock, p.ImageURL, p.CreatedAt, p.UpdatedAt); err != nil {
		return Product{}, fmt.Errorf("create product: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]Product, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	const query = `
		SELECT id, name, description, price_kzt, stock, COALESCE(image_url,''), created_at, updated_at
		FROM products
		WHERE $1 = '' OR name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, filter.Query, filter.Limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceKZT, &p.Stock, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Product, error) {
	const query = `SELECT id, name, description, price_kzt, stock, COALESCE(image_url,''), created_at, updated_at FROM products WHERE id=$1`
	var p Product
	if err := r.db.QueryRow(ctx, query, id).Scan(&p.ID, &p.Name, &p.Description, &p.PriceKZT, &p.Stock, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		return Product{}, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (r *PostgresRepository) Update(ctx context.Context, p Product) (Product, error) {
	const query = `
		UPDATE products
		SET name=$2, description=$3, price_kzt=$4, stock=$5, image_url=$6, updated_at=$7
		WHERE id=$1`
	tag, err := r.db.Exec(ctx, query, p.ID, p.Name, p.Description, p.PriceKZT, p.Stock, p.ImageURL, p.UpdatedAt)
	if err != nil {
		return Product{}, fmt.Errorf("update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Product{}, ErrNotFound
	}
	return p, nil
}

func (r *PostgresRepository) AddImage(ctx context.Context, image ProductImage) (ProductImage, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ProductImage{}, fmt.Errorf("begin image tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO product_images (id, product_id, object_name, url, created_at) VALUES ($1,$2,$3,$4,$5)`, image.ID, image.ProductID, image.Object, image.URL, image.CreatedAt); err != nil {
		return ProductImage{}, fmt.Errorf("insert image: %w", err)
	}
	tag, err := tx.Exec(ctx, `UPDATE products SET image_url=$2, updated_at=$3 WHERE id=$1`, image.ProductID, image.URL, time.Now().UTC())
	if err != nil {
		return ProductImage{}, fmt.Errorf("update image url: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ProductImage{}, ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return ProductImage{}, fmt.Errorf("commit image tx: %w", err)
	}
	return image, nil
}

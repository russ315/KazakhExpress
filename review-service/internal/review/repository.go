package review

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("review not found")

type Repository interface {
	Create(ctx context.Context, r Review) (Review, error)
	GetByID(ctx context.Context, id string) (Review, error)
	ListByProduct(ctx context.Context, filter ListFilter) ([]Review, error)
	Update(ctx context.Context, r Review) (Review, error)
	Delete(ctx context.Context, id string) error
	Rating(ctx context.Context, productID string) (Rating, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, review Review) (Review, error) {
	const query = `INSERT INTO reviews (id, product_id, customer_id, rating, comment, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`
	if _, err := r.db.Exec(ctx, query, review.ID, review.ProductID, review.CustomerID, review.Rating, review.Comment, review.CreatedAt, review.UpdatedAt); err != nil {
		return Review{}, fmt.Errorf("create review: %w", err)
	}
	return review, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (Review, error) {
	const query = `SELECT id, product_id, customer_id, rating, comment, created_at, updated_at FROM reviews WHERE id=$1`
	return r.scanOne(ctx, query, id)
}

func (r *PostgresRepository) ListByProduct(ctx context.Context, filter ListFilter) ([]Review, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	rows, err := r.db.Query(ctx, `SELECT id, product_id, customer_id, rating, comment, created_at, updated_at FROM reviews WHERE product_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, filter.ProductID, filter.Limit, filter.Offset)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()
	var reviews []Review
	for rows.Next() {
		var review Review
		if err := rows.Scan(&review.ID, &review.ProductID, &review.CustomerID, &review.Rating, &review.Comment, &review.CreatedAt, &review.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, review Review) (Review, error) {
	tag, err := r.db.Exec(ctx, `UPDATE reviews SET rating=$2, comment=$3, updated_at=$4 WHERE id=$1`, review.ID, review.Rating, review.Comment, review.UpdatedAt)
	if err != nil {
		return Review{}, fmt.Errorf("update review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Review{}, ErrNotFound
	}
	return review, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM reviews WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Rating(ctx context.Context, productID string) (Rating, error) {
	var rating Rating
	rating.ProductID = productID
	if err := r.db.QueryRow(ctx, `SELECT COALESCE(AVG(rating),0), COUNT(*) FROM reviews WHERE product_id=$1`, productID).Scan(&rating.Average, &rating.Count); err != nil {
		return Rating{}, fmt.Errorf("rating: %w", err)
	}
	return rating, nil
}

func (r *PostgresRepository) scanOne(ctx context.Context, query string, args ...any) (Review, error) {
	var review Review
	if err := r.db.QueryRow(ctx, query, args...).Scan(&review.ID, &review.ProductID, &review.CustomerID, &review.Rating, &review.Comment, &review.CreatedAt, &review.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Review{}, ErrNotFound
		}
		return Review{}, fmt.Errorf("get review: %w", err)
	}
	return review, nil
}

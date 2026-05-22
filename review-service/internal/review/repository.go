package review

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateReview(ctx context.Context, r Review) (Review, error)
	GetReview(ctx context.Context, id string) (Review, error)
	ListReviewsByProduct(ctx context.Context, productID string, page, pageSize int) (ListPage, error)
	UpdateReview(ctx context.Context, r Review) (Review, error)
	DeleteReview(ctx context.Context, id string) error

	GetProductRating(ctx context.Context, productID string) (ProductRating, error)
	UpsertProductRating(ctx context.Context, rating ProductRating) (ProductRating, error)

	GrantEligibility(ctx context.Context, userID, productID, orderID string, at time.Time) error
	IsEligible(ctx context.Context, userID, productID, orderID string) (bool, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateReview(ctx context.Context, item Review) (Review, error) {
	const query = `
		insert into reviews (
			id, product_id, user_id, order_id, rating, body, helpful_count, created_at, updated_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.Exec(ctx, query,
		item.ID, item.ProductID, item.UserID, item.OrderID, item.Rating, item.Body,
		item.HelpfulCount, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Review{}, ErrDuplicateReview
		}
		return Review{}, fmt.Errorf("create review: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) GetReview(ctx context.Context, id string) (Review, error) {
	const query = `
		select id, product_id, user_id, order_id, rating, body, helpful_count, created_at, updated_at
		from reviews where id = $1`

	item, err := scanReview(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Review{}, ErrNotFound
	}
	if err != nil {
		return Review{}, fmt.Errorf("get review: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) ListReviewsByProduct(ctx context.Context, productID string, page, pageSize int) (ListPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	offset := (page - 1) * pageSize

	var total int
	if err := r.db.QueryRow(ctx, `select count(*) from reviews where product_id = $1`, productID).Scan(&total); err != nil {
		return ListPage{}, fmt.Errorf("count reviews: %w", err)
	}

	const query = `
		select id, product_id, user_id, order_id, rating, body, helpful_count, created_at, updated_at
		from reviews
		where product_id = $1
		order by created_at desc
		limit $2 offset $3`

	rows, err := r.db.Query(ctx, query, productID, pageSize, offset)
	if err != nil {
		return ListPage{}, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]Review, 0)
	for rows.Next() {
		item, err := scanReview(rows)
		if err != nil {
			return ListPage{}, err
		}
		reviews = append(reviews, item)
	}
	if err := rows.Err(); err != nil {
		return ListPage{}, fmt.Errorf("iterate reviews: %w", err)
	}

	return ListPage{Reviews: reviews, Page: page, PageSize: pageSize, Total: total}, nil
}

func (r *PostgresRepository) UpdateReview(ctx context.Context, item Review) (Review, error) {
	const query = `
		update reviews
		set rating = $2, body = $3, updated_at = $4
		where id = $1`

	tag, err := r.db.Exec(ctx, query, item.ID, item.Rating, item.Body, item.UpdatedAt)
	if err != nil {
		return Review{}, fmt.Errorf("update review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Review{}, ErrNotFound
	}
	return item, nil
}

func (r *PostgresRepository) DeleteReview(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `delete from reviews where id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) GetProductRating(ctx context.Context, productID string) (ProductRating, error) {
	const query = `
		select product_id, rating_avg, rating_count, updated_at
		from product_ratings where product_id = $1`

	rating, err := scanRating(r.db.QueryRow(ctx, query, productID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ProductRating{ProductID: productID}, nil
	}
	if err != nil {
		return ProductRating{}, fmt.Errorf("get product rating: %w", err)
	}
	return rating, nil
}

func (r *PostgresRepository) UpsertProductRating(ctx context.Context, rating ProductRating) (ProductRating, error) {
	const query = `
		insert into product_ratings (product_id, rating_avg, rating_count, updated_at)
		values ($1, $2, $3, $4)
		on conflict (product_id) do update
		set rating_avg = excluded.rating_avg,
			rating_count = excluded.rating_count,
			updated_at = excluded.updated_at`

	_, err := r.db.Exec(ctx, query, rating.ProductID, rating.RatingAvg, rating.RatingCount, rating.UpdatedAt)
	if err != nil {
		return ProductRating{}, fmt.Errorf("upsert product rating: %w", err)
	}
	return rating, nil
}

func (r *PostgresRepository) GrantEligibility(ctx context.Context, userID, productID, orderID string, at time.Time) error {
	const query = `
		insert into review_eligibility (user_id, product_id, order_id, created_at)
		values ($1, $2, $3, $4)
		on conflict do nothing`
	_, err := r.db.Exec(ctx, query, userID, productID, orderID, at)
	if err != nil {
		return fmt.Errorf("grant eligibility: %w", err)
	}
	return nil
}

func (r *PostgresRepository) IsEligible(ctx context.Context, userID, productID, orderID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		select exists(
			select 1 from review_eligibility
			where user_id = $1 and product_id = $2 and order_id = $3
		)`, userID, productID, orderID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check eligibility: %w", err)
	}
	return exists, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanReview(row rowScanner) (Review, error) {
	var item Review
	err := row.Scan(
		&item.ID, &item.ProductID, &item.UserID, &item.OrderID,
		&item.Rating, &item.Body, &item.HelpfulCount, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanRating(row rowScanner) (ProductRating, error) {
	var rating ProductRating
	err := row.Scan(&rating.ProductID, &rating.RatingAvg, &rating.RatingCount, &rating.UpdatedAt)
	return rating, err
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ Code() string }
	return errors.As(err, &pgErr) && pgErr.Code() == "23505"
}

// MemoryRepository for tests.
type MemoryRepository struct {
	reviews      map[string]Review
	eligibility  map[string]struct{}
	ratings      map[string]ProductRating
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		reviews:     make(map[string]Review),
		eligibility: make(map[string]struct{}),
		ratings:     make(map[string]ProductRating),
	}
}

func eligibilityKey(userID, productID, orderID string) string {
	return userID + "|" + productID + "|" + orderID
}

func (r *MemoryRepository) CreateReview(ctx context.Context, item Review) (Review, error) {
	for _, existing := range r.reviews {
		if existing.ProductID == item.ProductID && existing.UserID == item.UserID {
			return Review{}, ErrDuplicateReview
		}
	}
	r.reviews[item.ID] = item
	return item, nil
}

func (r *MemoryRepository) GetReview(ctx context.Context, id string) (Review, error) {
	item, ok := r.reviews[id]
	if !ok {
		return Review{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryRepository) ListReviewsByProduct(ctx context.Context, productID string, page, pageSize int) (ListPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	all := make([]Review, 0)
	for _, item := range r.reviews {
		if item.ProductID == productID {
			all = append(all, item)
		}
	}
	total := len(all)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return ListPage{Reviews: all[start:end], Page: page, PageSize: pageSize, Total: total}, nil
}

func (r *MemoryRepository) UpdateReview(ctx context.Context, item Review) (Review, error) {
	if _, ok := r.reviews[item.ID]; !ok {
		return Review{}, ErrNotFound
	}
	r.reviews[item.ID] = item
	return item, nil
}

func (r *MemoryRepository) DeleteReview(ctx context.Context, id string) error {
	if _, ok := r.reviews[id]; !ok {
		return ErrNotFound
	}
	delete(r.reviews, id)
	return nil
}

func (r *MemoryRepository) GetProductRating(ctx context.Context, productID string) (ProductRating, error) {
	rating, ok := r.ratings[productID]
	if !ok {
		return ProductRating{ProductID: productID}, nil
	}
	return rating, nil
}

func (r *MemoryRepository) UpsertProductRating(ctx context.Context, rating ProductRating) (ProductRating, error) {
	r.ratings[rating.ProductID] = rating
	return rating, nil
}

func (r *MemoryRepository) GrantEligibility(ctx context.Context, userID, productID, orderID string, at time.Time) error {
	r.eligibility[eligibilityKey(userID, productID, orderID)] = struct{}{}
	return nil
}

func (r *MemoryRepository) IsEligible(ctx context.Context, userID, productID, orderID string) (bool, error) {
	_, ok := r.eligibility[eligibilityKey(userID, productID, orderID)]
	return ok, nil
}

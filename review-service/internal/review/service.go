package review

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	repo      Repository
	cache     CacheService
	publisher EventPublisher
}

func NewService(repo Repository, cache CacheService, publisher EventPublisher) *Service {
	if cache == nil {
		cache = NoopCache{}
	}
	if publisher == nil {
		publisher = NoopPublisher{}
	}
	return &Service{repo: repo, cache: cache, publisher: publisher}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Review, error) {
	if input.ProductID == "" || input.UserID == "" || input.OrderID == "" || input.Rating < 1 || input.Rating > 5 {
		return Review{}, ErrInvalidInput
	}

	eligible, err := s.repo.IsEligible(ctx, input.UserID, input.ProductID, input.OrderID)
	if err != nil {
		return Review{}, err
	}
	if !eligible {
		return Review{}, ErrNotEligible
	}

	now := time.Now().UTC()
	item := Review{
		ID:           fmt.Sprintf("rev-%s-%s-%d", input.ProductID, input.UserID, now.UnixNano()),
		ProductID:    input.ProductID,
		UserID:       input.UserID,
		OrderID:      input.OrderID,
		Rating:       input.Rating,
		Body:         input.Body,
		HelpfulCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := s.repo.CreateReview(ctx, item)
	if err != nil {
		return Review{}, err
	}

	rating, err := s.recomputeRating(ctx, input.ProductID)
	if err != nil {
		return Review{}, err
	}

	s.invalidateProductCaches(ctx, input.ProductID)
	_ = s.publisher.PublishReviewCreated(ctx, ReviewEvent{
		ReviewID:  created.ID,
		ProductID: created.ProductID,
		UserID:    created.UserID,
		Rating:    created.Rating,
		Timestamp: now,
	})
	_ = s.publisher.PublishProductRatingUpdated(ctx, ratingEvent(rating, now))

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Review, error) {
	if id == "" {
		return Review{}, ErrInvalidInput
	}
	return s.repo.GetReview(ctx, id)
}

func (s *Service) ListByProduct(ctx context.Context, productID string, page, pageSize int) (ListPage, error) {
	if productID == "" {
		return ListPage{}, ErrInvalidInput
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}

	if cached, ok, err := s.cache.GetReviewsPage(ctx, productID, page); err == nil && ok {
		return cached, nil
	}

	result, err := s.repo.ListReviewsByProduct(ctx, productID, page, pageSize)
	if err != nil {
		return ListPage{}, err
	}

	_ = s.cache.SetReviewsPage(ctx, productID, result, 5*time.Minute)
	return result, nil
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Review, error) {
	if id == "" {
		return Review{}, ErrInvalidInput
	}

	item, err := s.repo.GetReview(ctx, id)
	if err != nil {
		return Review{}, err
	}

	if input.Rating != nil {
		if *input.Rating < 1 || *input.Rating > 5 {
			return Review{}, ErrInvalidInput
		}
		item.Rating = *input.Rating
	}
	if input.Body != nil {
		item.Body = *input.Body
	}
	item.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.UpdateReview(ctx, item)
	if err != nil {
		return Review{}, err
	}

	rating, err := s.recomputeRating(ctx, item.ProductID)
	if err != nil {
		return Review{}, err
	}

	now := updated.UpdatedAt
	s.invalidateProductCaches(ctx, item.ProductID)
	_ = s.publisher.PublishReviewUpdated(ctx, ReviewEvent{
		ReviewID:  updated.ID,
		ProductID: updated.ProductID,
		UserID:    updated.UserID,
		Rating:    updated.Rating,
		Timestamp: now,
	})
	_ = s.publisher.PublishProductRatingUpdated(ctx, ratingEvent(rating, now))

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidInput
	}

	item, err := s.repo.GetReview(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteReview(ctx, id); err != nil {
		return err
	}

	rating, err := s.recomputeRating(ctx, item.ProductID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	s.invalidateProductCaches(ctx, item.ProductID)
	_ = s.publisher.PublishReviewDeleted(ctx, ReviewEvent{
		ReviewID:  item.ID,
		ProductID: item.ProductID,
		UserID:    item.UserID,
		Rating:    item.Rating,
		Timestamp: now,
	})
	_ = s.publisher.PublishProductRatingUpdated(ctx, ratingEvent(rating, now))

	return nil
}

func (s *Service) GetProductRating(ctx context.Context, productID string) (ProductRating, error) {
	if productID == "" {
		return ProductRating{}, ErrInvalidInput
	}

	if cached, ok, err := s.cache.GetRating(ctx, productID); err == nil && ok {
		return cached, nil
	}

	rating, err := s.repo.GetProductRating(ctx, productID)
	if err != nil {
		return ProductRating{}, err
	}

	_ = s.cache.SetRating(ctx, rating, 5*time.Minute)
	return rating, nil
}

func (s *Service) HandleOrderCompleted(ctx context.Context, event OrderCompletedEvent) error {
	if event.OrderID == "" || event.CustomerID == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC()
	for _, item := range event.Items {
		if item.ProductID == "" {
			continue
		}
		if err := s.repo.GrantEligibility(ctx, event.CustomerID, item.ProductID, event.OrderID, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) recomputeRating(ctx context.Context, productID string) (ProductRating, error) {
	page, err := s.repo.ListReviewsByProduct(ctx, productID, 1, 10000)
	if err != nil {
		return ProductRating{}, err
	}

	var sum int
	for _, item := range page.Reviews {
		sum += item.Rating
	}

	count := len(page.Reviews)
	avg := 0.0
	if count > 0 {
		avg = float64(sum) / float64(count)
	}

	now := time.Now().UTC()
	rating := ProductRating{
		ProductID:   productID,
		RatingAvg:   avg,
		RatingCount: count,
		UpdatedAt:   now,
	}

	updated, err := s.repo.UpsertProductRating(ctx, rating)
	if err != nil {
		return ProductRating{}, err
	}

	_ = s.cache.SetRating(ctx, updated, 5*time.Minute)
	return updated, nil
}

func (s *Service) invalidateProductCaches(ctx context.Context, productID string) {
	_ = s.cache.InvalidateRating(ctx, productID)
	_ = s.cache.InvalidateReviews(ctx, productID)
}

func ratingEvent(rating ProductRating, at time.Time) RatingEvent {
	return RatingEvent{
		ProductID:   rating.ProductID,
		RatingAvg:   rating.RatingAvg,
		RatingCount: rating.RatingCount,
		Timestamp:   at,
	}
}

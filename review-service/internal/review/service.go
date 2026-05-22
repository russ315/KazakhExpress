package review

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var ErrInvalidInput = errors.New("invalid review input")

type Cache interface {
	GetRating(ctx context.Context, productID string) (Rating, bool, error)
	SetRating(ctx context.Context, rating Rating) error
	DeleteRating(ctx context.Context, productID string) error
}

type EventPublisher interface {
	PublishReviewCreated(ctx context.Context, event Event) error
	PublishReviewUpdated(ctx context.Context, event Event) error
	PublishReviewDeleted(ctx context.Context, event Event) error
	PublishRatingUpdated(ctx context.Context, rating Rating) error
}

type Service struct {
	repo      Repository
	cache     Cache
	publisher EventPublisher
}

func NewService(repo Repository, cache Cache, publisher EventPublisher) *Service {
	return &Service{repo: repo, cache: cache, publisher: publisher}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Review, error) {
	if input.ProductID == "" || input.CustomerID == "" || input.Rating < 1 || input.Rating > 5 {
		return Review{}, ErrInvalidInput
	}
	now := time.Now().UTC()
	review := Review{ID: fmt.Sprintf("rev-%d", now.UnixNano()), ProductID: input.ProductID, CustomerID: input.CustomerID, Rating: input.Rating, Comment: input.Comment, CreatedAt: now, UpdatedAt: now}

	dbStart := time.Now()
	created, err := s.repo.Create(ctx, review)
	if err != nil {
		ReviewDBOperationsTotal.WithLabelValues("create", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("create").Observe(time.Since(dbStart).Seconds())
		return Review{}, err
	}
	ReviewDBOperationsTotal.WithLabelValues("create", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("create").Observe(time.Since(dbStart).Seconds())

	ReviewsCreatedTotal.WithLabelValues(strconv.Itoa(created.Rating)).Inc()
	RecordRatingLeft(created.Rating)

	s.ratingChanged(ctx, created.ProductID)
	if s.publisher != nil {
		_ = s.publisher.PublishReviewCreated(ctx, Event{ReviewID: created.ID, ProductID: created.ProductID, CustomerID: created.CustomerID, Rating: created.Rating, OccurredAt: now})
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, id string) (Review, error) {
	if id == "" {
		return Review{}, ErrInvalidInput
	}
	dbStart := time.Now()
	res, err := s.repo.GetByID(ctx, id)
	if err != nil {
		ReviewDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStart).Seconds())
		return Review{}, err
	}
	ReviewDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStart).Seconds())
	return res, nil
}

func (s *Service) ListByProduct(ctx context.Context, filter ListFilter) ([]Review, error) {
	if filter.ProductID == "" {
		return nil, ErrInvalidInput
	}
	dbStart := time.Now()
	res, err := s.repo.ListByProduct(ctx, filter)
	if err != nil {
		ReviewDBOperationsTotal.WithLabelValues("list_by_product", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("list_by_product").Observe(time.Since(dbStart).Seconds())
		return nil, err
	}
	ReviewDBOperationsTotal.WithLabelValues("list_by_product", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("list_by_product").Observe(time.Since(dbStart).Seconds())
	return res, nil
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Review, error) {
	if id == "" || input.Rating < 1 || input.Rating > 5 {
		return Review{}, ErrInvalidInput
	}
	dbStartGet := time.Now()
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		ReviewDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())
		return Review{}, err
	}
	ReviewDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())

	review.Rating = input.Rating
	review.Comment = input.Comment
	review.UpdatedAt = time.Now().UTC()

	dbStartUpdate := time.Now()
	updated, err := s.repo.Update(ctx, review)
	if err != nil {
		ReviewDBOperationsTotal.WithLabelValues("update", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())
		return Review{}, err
	}
	ReviewDBOperationsTotal.WithLabelValues("update", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("update").Observe(time.Since(dbStartUpdate).Seconds())

	RecordRatingLeft(updated.Rating)

	s.ratingChanged(ctx, updated.ProductID)
	if s.publisher != nil {
		_ = s.publisher.PublishReviewUpdated(ctx, Event{ReviewID: updated.ID, ProductID: updated.ProductID, Rating: updated.Rating, OccurredAt: updated.UpdatedAt})
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	dbStartGet := time.Now()
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		ReviewDBOperationsTotal.WithLabelValues("get_by_id", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())
		return err
	}
	ReviewDBOperationsTotal.WithLabelValues("get_by_id", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("get_by_id").Observe(time.Since(dbStartGet).Seconds())

	dbStartDel := time.Now()
	if err := s.repo.Delete(ctx, id); err != nil {
		ReviewDBOperationsTotal.WithLabelValues("delete", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("delete").Observe(time.Since(dbStartDel).Seconds())
		return err
	}
	ReviewDBOperationsTotal.WithLabelValues("delete", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("delete").Observe(time.Since(dbStartDel).Seconds())

	ReviewsDeletedTotal.Inc()

	s.ratingChanged(ctx, review.ProductID)
	if s.publisher != nil {
		_ = s.publisher.PublishReviewDeleted(ctx, Event{ReviewID: id, ProductID: review.ProductID, OccurredAt: time.Now().UTC()})
	}
	return nil
}

func (s *Service) Rating(ctx context.Context, productID string) (Rating, error) {
	if productID == "" {
		return Rating{}, ErrInvalidInput
	}
	if s.cache != nil {
		if rating, ok, err := s.cache.GetRating(ctx, productID); err == nil && ok {
			return rating, nil
		}
	}
	dbStart := time.Now()
	rating, err := s.repo.Rating(ctx, productID)
	if err != nil {
		ReviewDBOperationsTotal.WithLabelValues("rating", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("rating").Observe(time.Since(dbStart).Seconds())
		return Rating{}, err
	}
	ReviewDBOperationsTotal.WithLabelValues("rating", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("rating").Observe(time.Since(dbStart).Seconds())

	if s.cache != nil {
		_ = s.cache.SetRating(ctx, rating)
	}
	return rating, nil
}

func (s *Service) ratingChanged(ctx context.Context, productID string) {
	if s.cache != nil {
		_ = s.cache.DeleteRating(ctx, productID)
	}
	dbStart := time.Now()
	rating, err := s.repo.Rating(ctx, productID)
	if err != nil {
		ReviewDBOperationsTotal.WithLabelValues("rating_changed", "error").Inc()
		ReviewDBOperationDurationSeconds.WithLabelValues("rating_changed").Observe(time.Since(dbStart).Seconds())
		return
	}
	ReviewDBOperationsTotal.WithLabelValues("rating_changed", "success").Inc()
	ReviewDBOperationDurationSeconds.WithLabelValues("rating_changed").Observe(time.Since(dbStart).Seconds())

	if s.publisher != nil {
		_ = s.publisher.PublishRatingUpdated(ctx, rating)
	}
}

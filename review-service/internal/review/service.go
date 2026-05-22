package review

import (
	"context"
	"errors"
	"fmt"
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
	created, err := s.repo.Create(ctx, review)
	if err != nil {
		return Review{}, err
	}
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
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByProduct(ctx context.Context, filter ListFilter) ([]Review, error) {
	if filter.ProductID == "" {
		return nil, ErrInvalidInput
	}
	return s.repo.ListByProduct(ctx, filter)
}

func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Review, error) {
	if id == "" || input.Rating < 1 || input.Rating > 5 {
		return Review{}, ErrInvalidInput
	}
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Review{}, err
	}
	review.Rating = input.Rating
	review.Comment = input.Comment
	review.UpdatedAt = time.Now().UTC()
	updated, err := s.repo.Update(ctx, review)
	if err != nil {
		return Review{}, err
	}
	s.ratingChanged(ctx, updated.ProductID)
	if s.publisher != nil {
		_ = s.publisher.PublishReviewUpdated(ctx, Event{ReviewID: updated.ID, ProductID: updated.ProductID, Rating: updated.Rating, OccurredAt: updated.UpdatedAt})
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	review, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
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
	rating, err := s.repo.Rating(ctx, productID)
	if err != nil {
		return Rating{}, err
	}
	if s.cache != nil {
		_ = s.cache.SetRating(ctx, rating)
	}
	return rating, nil
}

func (s *Service) ratingChanged(ctx context.Context, productID string) {
	if s.cache != nil {
		_ = s.cache.DeleteRating(ctx, productID)
	}
	rating, err := s.repo.Rating(ctx, productID)
	if err == nil && s.publisher != nil {
		_ = s.publisher.PublishRatingUpdated(ctx, rating)
	}
}

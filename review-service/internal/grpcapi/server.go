package grpcapi

import (
	"context"
	"errors"
	"time"

	"kazakhexpress/review-service/internal/review"
	reviewv1 "kazakhexpress/review-service/internal/reviewv1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	reviewv1.UnimplementedReviewServiceServer
	service *review.Service
}

func NewServer(service *review.Service) *Server {
	return &Server{service: service}
}

func (s *Server) CreateReview(ctx context.Context, req *reviewv1.CreateReviewRequest) (*reviewv1.Review, error) {
	result, err := s.service.Create(ctx, review.CreateInput{
		ProductID: req.GetProductId(),
		UserID:    req.GetUserId(),
		OrderID:   req.GetOrderId(),
		Rating:    int(req.GetRating()),
		Body:      req.GetBody(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return reviewToProto(result), nil
}

func (s *Server) GetReview(ctx context.Context, req *reviewv1.GetReviewRequest) (*reviewv1.Review, error) {
	result, err := s.service.GetByID(ctx, req.GetReviewId())
	if err != nil {
		return nil, mapError(err)
	}
	return reviewToProto(result), nil
}

func (s *Server) ListProductReviews(ctx context.Context, req *reviewv1.ListProductReviewsRequest) (*reviewv1.ListProductReviewsResponse, error) {
	page, err := s.service.ListByProduct(ctx, req.GetProductId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, mapError(err)
	}
	reviews := make([]*reviewv1.Review, 0, len(page.Reviews))
	for _, item := range page.Reviews {
		reviews = append(reviews, reviewToProto(item))
	}
	return &reviewv1.ListProductReviewsResponse{
		Reviews:  reviews,
		Page:     int32(page.Page),
		PageSize: int32(page.PageSize),
		Total:    int32(page.Total),
	}, nil
}

func (s *Server) UpdateReview(ctx context.Context, req *reviewv1.UpdateReviewRequest) (*reviewv1.Review, error) {
	input := review.UpdateInput{}
	if req.Rating != nil {
		rating := int(*req.Rating)
		input.Rating = &rating
	}
	if req.Body != nil {
		body := *req.Body
		input.Body = &body
	}
	result, err := s.service.Update(ctx, req.GetReviewId(), input)
	if err != nil {
		return nil, mapError(err)
	}
	return reviewToProto(result), nil
}

func (s *Server) DeleteReview(ctx context.Context, req *reviewv1.DeleteReviewRequest) (*reviewv1.DeleteReviewResponse, error) {
	if err := s.service.Delete(ctx, req.GetReviewId()); err != nil {
		return nil, mapError(err)
	}
	return &reviewv1.DeleteReviewResponse{}, nil
}

func (s *Server) GetProductRating(ctx context.Context, req *reviewv1.GetProductRatingRequest) (*reviewv1.ProductRating, error) {
	result, err := s.service.GetProductRating(ctx, req.GetProductId())
	if err != nil {
		return nil, mapError(err)
	}
	return ratingToProto(result), nil
}

func reviewToProto(item review.Review) *reviewv1.Review {
	return &reviewv1.Review{
		Id:           item.ID,
		ProductId:    item.ProductID,
		UserId:       item.UserID,
		OrderId:      item.OrderID,
		Rating:       int32(item.Rating),
		Body:         item.Body,
		HelpfulCount: int32(item.HelpfulCount),
		CreatedAt:    formatTime(item.CreatedAt),
		UpdatedAt:    formatTime(item.UpdatedAt),
	}
}

func ratingToProto(item review.ProductRating) *reviewv1.ProductRating {
	return &reviewv1.ProductRating{
		ProductId:   item.ProductID,
		RatingAvg:   item.RatingAvg,
		RatingCount: int32(item.RatingCount),
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func mapError(err error) error {
	switch {
	case errors.Is(err, review.ErrInvalidInput), errors.Is(err, review.ErrNotEligible):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, review.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, review.ErrDuplicateReview):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

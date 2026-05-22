package grpcapi

import (
	"context"
	"time"

	"kazakhexpress/review-service/internal/review"

	reviewv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/review/v1"
)

type Server struct {
	reviewv1.UnimplementedReviewServiceServer
	service *review.Service
}

func NewServer(service *review.Service) *Server {
	return &Server{service: service}
}

func (s *Server) HealthCheck(ctx context.Context, req *reviewv1.HealthCheckRequest) (*reviewv1.HealthCheckResponse, error) {
	return &reviewv1.HealthCheckResponse{Status: "ok"}, nil
}

func (s *Server) CreateReview(ctx context.Context, req *reviewv1.CreateReviewRequest) (*reviewv1.Review, error) {
	created, err := s.service.Create(ctx, review.CreateInput{ProductID: req.GetProductId(), CustomerID: req.GetCustomerId(), Rating: int(req.GetRating()), Comment: req.GetComment()})
	if err != nil {
		return nil, err
	}
	return toProto(created), nil
}

func (s *Server) GetReview(ctx context.Context, req *reviewv1.GetReviewRequest) (*reviewv1.Review, error) {
	found, err := s.service.Get(ctx, req.GetReviewId())
	if err != nil {
		return nil, err
	}
	return toProto(found), nil
}

func (s *Server) ListProductReviews(ctx context.Context, req *reviewv1.ListProductReviewsRequest) (*reviewv1.ListProductReviewsResponse, error) {
	reviews, err := s.service.ListByProduct(ctx, review.ListFilter{ProductID: req.GetProductId(), Limit: int(req.GetLimit()), Offset: int(req.GetOffset())})
	if err != nil {
		return nil, err
	}
	out := make([]*reviewv1.Review, 0, len(reviews))
	for _, item := range reviews {
		out = append(out, toProto(item))
	}
	return &reviewv1.ListProductReviewsResponse{Reviews: out}, nil
}

func (s *Server) UpdateReview(ctx context.Context, req *reviewv1.UpdateReviewRequest) (*reviewv1.Review, error) {
	updated, err := s.service.Update(ctx, req.GetReviewId(), review.UpdateInput{Rating: int(req.GetRating()), Comment: req.GetComment()})
	if err != nil {
		return nil, err
	}
	return toProto(updated), nil
}

func (s *Server) DeleteReview(ctx context.Context, req *reviewv1.DeleteReviewRequest) (*reviewv1.DeleteReviewResponse, error) {
	if err := s.service.Delete(ctx, req.GetReviewId()); err != nil {
		return nil, err
	}
	return &reviewv1.DeleteReviewResponse{Deleted: true}, nil
}

func (s *Server) GetProductRating(ctx context.Context, req *reviewv1.GetProductRatingRequest) (*reviewv1.ProductRating, error) {
	rating, err := s.service.Rating(ctx, req.GetProductId())
	if err != nil {
		return nil, err
	}
	return &reviewv1.ProductRating{ProductId: rating.ProductID, AverageRating: rating.Average, ReviewCount: rating.Count}, nil
}

func toProto(r review.Review) *reviewv1.Review {
	return &reviewv1.Review{
		Id:         r.ID,
		ProductId:  r.ProductID,
		CustomerId: r.CustomerID,
		Rating:     int32(r.Rating),
		Comment:    r.Comment,
		CreatedAt:  formatTime(r.CreatedAt),
		UpdatedAt:  formatTime(r.UpdatedAt),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

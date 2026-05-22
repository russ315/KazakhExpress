package reviewservice

import (
	"context"
	"fmt"

	reviewv1 "kazakhexpress/review-service/internal/reviewv1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client interface {
	Health(ctx context.Context) error
	CreateReview(ctx context.Context, productID string, input CreateReviewRequest) (Review, error)
	GetReview(ctx context.Context, reviewID string) (Review, error)
	ListProductReviews(ctx context.Context, productID string, page, pageSize int) (ListReviewsResponse, error)
	UpdateReview(ctx context.Context, reviewID string, input UpdateReviewRequest) (Review, error)
	DeleteReview(ctx context.Context, reviewID string) error
	GetProductRating(ctx context.Context, productID string) (ProductRating, error)
}

type GRPCClient struct {
	conn   *grpc.ClientConn
	client reviewv1.ReviewServiceClient
}

func NewGRPCClient(target string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create review grpc client: %w", err)
	}
	return &GRPCClient{
		conn:   conn,
		client: reviewv1.NewReviewServiceClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCClient) Health(ctx context.Context) error {
	_, err := c.client.GetProductRating(ctx, &reviewv1.GetProductRatingRequest{ProductId: "_health"})
	return err
}

func (c *GRPCClient) CreateReview(ctx context.Context, productID string, input CreateReviewRequest) (Review, error) {
	out, err := c.client.CreateReview(ctx, &reviewv1.CreateReviewRequest{
		ProductId: productID,
		UserId:    input.UserID,
		OrderId:   input.OrderID,
		Rating:    int32(input.Rating),
		Body:      input.Body,
	})
	return reviewFromProto(out), err
}

func (c *GRPCClient) GetReview(ctx context.Context, reviewID string) (Review, error) {
	out, err := c.client.GetReview(ctx, &reviewv1.GetReviewRequest{ReviewId: reviewID})
	return reviewFromProto(out), err
}

func (c *GRPCClient) ListProductReviews(ctx context.Context, productID string, page, pageSize int) (ListReviewsResponse, error) {
	out, err := c.client.ListProductReviews(ctx, &reviewv1.ListProductReviewsRequest{
		ProductId: productID,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		return ListReviewsResponse{}, err
	}
	reviews := make([]Review, 0, len(out.GetReviews()))
	for _, item := range out.GetReviews() {
		reviews = append(reviews, reviewFromProto(item))
	}
	return ListReviewsResponse{
		Reviews:  reviews,
		Page:     int(out.GetPage()),
		PageSize: int(out.GetPageSize()),
		Total:    int(out.GetTotal()),
	}, nil
}

func (c *GRPCClient) UpdateReview(ctx context.Context, reviewID string, input UpdateReviewRequest) (Review, error) {
	req := &reviewv1.UpdateReviewRequest{ReviewId: reviewID}
	if input.Rating != nil {
		rating := int32(*input.Rating)
		req.Rating = &rating
	}
	if input.Body != nil {
		req.Body = input.Body
	}
	out, err := c.client.UpdateReview(ctx, req)
	return reviewFromProto(out), err
}

func (c *GRPCClient) DeleteReview(ctx context.Context, reviewID string) error {
	_, err := c.client.DeleteReview(ctx, &reviewv1.DeleteReviewRequest{ReviewId: reviewID})
	return err
}

func (c *GRPCClient) GetProductRating(ctx context.Context, productID string) (ProductRating, error) {
	out, err := c.client.GetProductRating(ctx, &reviewv1.GetProductRatingRequest{ProductId: productID})
	if err != nil {
		return ProductRating{}, err
	}
	return ProductRating{
		ProductID:   out.GetProductId(),
		RatingAvg:   out.GetRatingAvg(),
		RatingCount: int(out.GetRatingCount()),
	}, nil
}

func reviewFromProto(item *reviewv1.Review) Review {
	if item == nil {
		return Review{}
	}
	return Review{
		ID:           item.GetId(),
		ProductID:    item.GetProductId(),
		UserID:       item.GetUserId(),
		OrderID:      item.GetOrderId(),
		Rating:       int(item.GetRating()),
		Body:         item.GetBody(),
		HelpfulCount: int(item.GetHelpfulCount()),
		CreatedAt:    item.GetCreatedAt(),
		UpdatedAt:    item.GetUpdatedAt(),
	}
}

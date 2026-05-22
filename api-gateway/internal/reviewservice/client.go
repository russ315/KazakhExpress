package reviewservice

import (
	"context"
	"fmt"

	reviewv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/review/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client interface {
	Health(ctx context.Context) error
	Create(ctx context.Context, productID string, input CreateReviewRequest) (Review, error)
	Get(ctx context.Context, reviewID string) (Review, error)
	List(ctx context.Context, productID string, limit, offset int) ([]Review, error)
	Update(ctx context.Context, reviewID string, input UpdateReviewRequest) (Review, error)
	Delete(ctx context.Context, reviewID string) error
	Rating(ctx context.Context, productID string) (Rating, error)
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
	return &GRPCClient{conn: conn, client: reviewv1.NewReviewServiceClient(conn)}, nil
}

func (c *GRPCClient) Close() error { return c.conn.Close() }

func (c *GRPCClient) Health(ctx context.Context) error {
	_, err := c.client.HealthCheck(ctx, &reviewv1.HealthCheckRequest{})
	return err
}

func (c *GRPCClient) Create(ctx context.Context, productID string, input CreateReviewRequest) (Review, error) {
	output, err := c.client.CreateReview(ctx, &reviewv1.CreateReviewRequest{ProductId: productID, CustomerId: input.CustomerID, Rating: int32(input.Rating), Comment: input.Comment})
	return fromProto(output), err
}

func (c *GRPCClient) Get(ctx context.Context, reviewID string) (Review, error) {
	output, err := c.client.GetReview(ctx, &reviewv1.GetReviewRequest{ReviewId: reviewID})
	return fromProto(output), err
}

func (c *GRPCClient) List(ctx context.Context, productID string, limit, offset int) ([]Review, error) {
	output, err := c.client.ListProductReviews(ctx, &reviewv1.ListProductReviewsRequest{ProductId: productID, Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, err
	}
	reviews := make([]Review, 0, len(output.GetReviews()))
	for _, item := range output.GetReviews() {
		reviews = append(reviews, fromProto(item))
	}
	return reviews, nil
}

func (c *GRPCClient) Update(ctx context.Context, reviewID string, input UpdateReviewRequest) (Review, error) {
	output, err := c.client.UpdateReview(ctx, &reviewv1.UpdateReviewRequest{ReviewId: reviewID, Rating: int32(input.Rating), Comment: input.Comment})
	return fromProto(output), err
}

func (c *GRPCClient) Delete(ctx context.Context, reviewID string) error {
	_, err := c.client.DeleteReview(ctx, &reviewv1.DeleteReviewRequest{ReviewId: reviewID})
	return err
}

func (c *GRPCClient) Rating(ctx context.Context, productID string) (Rating, error) {
	output, err := c.client.GetProductRating(ctx, &reviewv1.GetProductRatingRequest{ProductId: productID})
	if output == nil {
		return Rating{}, err
	}
	return Rating{ProductID: output.GetProductId(), Average: output.GetAverageRating(), Count: output.GetReviewCount()}, err
}

func fromProto(item *reviewv1.Review) Review {
	if item == nil {
		return Review{}
	}
	return Review{ID: item.GetId(), ProductID: item.GetProductId(), CustomerID: item.GetCustomerId(), Rating: int(item.GetRating()), Comment: item.GetComment(), CreatedAt: item.GetCreatedAt(), UpdatedAt: item.GetUpdatedAt()}
}

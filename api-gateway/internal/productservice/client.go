package productservice

import (
	"context"
	"fmt"

	productv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/product/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client interface {
	Health(ctx context.Context) error
	Create(ctx context.Context, input CreateProductRequest) (Product, error)
	Get(ctx context.Context, productID string) (Product, error)
	List(ctx context.Context, limit, offset int, query string) ([]Product, error)
	UpdateStock(ctx context.Context, productID string, stock int) (Product, error)
	AddImage(ctx context.Context, productID, filename, contentType string, content []byte) (ProductImage, error)
}

type GRPCClient struct {
	conn   *grpc.ClientConn
	client productv1.ProductServiceClient
}

func NewGRPCClient(target string) (*GRPCClient, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create product grpc client: %w", err)
	}
	return &GRPCClient{conn: conn, client: productv1.NewProductServiceClient(conn)}, nil
}

func (c *GRPCClient) Close() error { return c.conn.Close() }

func (c *GRPCClient) Health(ctx context.Context) error {
	_, err := c.client.HealthCheck(ctx, &productv1.HealthCheckRequest{})
	return err
}

func (c *GRPCClient) Create(ctx context.Context, input CreateProductRequest) (Product, error) {
	output, err := c.client.CreateProduct(ctx, &productv1.CreateProductRequest{Name: input.Name, Description: input.Description, PriceKzt: input.PriceKZT, Stock: int32(input.Stock)})
	return fromProto(output), err
}

func (c *GRPCClient) Get(ctx context.Context, productID string) (Product, error) {
	output, err := c.client.GetProduct(ctx, &productv1.GetProductRequest{ProductId: productID})
	return fromProto(output), err
}

func (c *GRPCClient) List(ctx context.Context, limit, offset int, query string) ([]Product, error) {
	output, err := c.client.ListProducts(ctx, &productv1.ListProductsRequest{Limit: int32(limit), Offset: int32(offset), Query: query})
	if err != nil {
		return nil, err
	}
	products := make([]Product, 0, len(output.GetProducts()))
	for _, item := range output.GetProducts() {
		products = append(products, fromProto(item))
	}
	return products, nil
}

func (c *GRPCClient) UpdateStock(ctx context.Context, productID string, stock int) (Product, error) {
	output, err := c.client.UpdateStock(ctx, &productv1.UpdateStockRequest{ProductId: productID, Stock: int32(stock)})
	return fromProto(output), err
}

func (c *GRPCClient) AddImage(ctx context.Context, productID, filename, contentType string, content []byte) (ProductImage, error) {
	output, err := c.client.AddProductImage(ctx, &productv1.AddProductImageRequest{ProductId: productID, Filename: filename, ContentType: contentType, Content: content})
	if output == nil {
		return ProductImage{}, err
	}
	return ProductImage{ID: output.GetId(), ProductID: output.GetProductId(), Object: output.GetObjectName(), URL: output.GetUrl(), CreatedAt: output.GetCreatedAt()}, err
}

func fromProto(item *productv1.Product) Product {
	if item == nil {
		return Product{}
	}
	return Product{ID: item.GetId(), Name: item.GetName(), Description: item.GetDescription(), PriceKZT: item.GetPriceKzt(), Stock: int(item.GetStock()), ImageURL: item.GetImageUrl(), CreatedAt: item.GetCreatedAt(), UpdatedAt: item.GetUpdatedAt()}
}

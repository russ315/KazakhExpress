package grpcapi

import (
	"context"
	"time"

	"kazakhexpress/product-service/internal/product"

	productv1 "github.com/maqsatto/kazakhexpress-proto/gen/go/kazakhexpress/product/v1"
)

type Server struct {
	productv1.UnimplementedProductServiceServer
	service *product.Service
}

func NewServer(service *product.Service) *Server {
	return &Server{service: service}
}

func (s *Server) HealthCheck(ctx context.Context, req *productv1.HealthCheckRequest) (*productv1.HealthCheckResponse, error) {
	return &productv1.HealthCheckResponse{Status: "ok"}, nil
}

func (s *Server) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.Product, error) {
	p, err := s.service.Create(ctx, product.CreateInput{Name: req.GetName(), Description: req.GetDescription(), PriceKZT: req.GetPriceKzt(), Stock: int(req.GetStock())})
	if err != nil {
		return nil, err
	}
	return toProto(p), nil
}

func (s *Server) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.Product, error) {
	p, err := s.service.GetByID(ctx, req.GetProductId())
	if err != nil {
		return nil, err
	}
	return toProto(p), nil
}

func (s *Server) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	products, err := s.service.List(ctx, product.ListFilter{Limit: int(req.GetLimit()), Offset: int(req.GetOffset()), Query: req.GetQuery()})
	if err != nil {
		return nil, err
	}
	out := make([]*productv1.Product, 0, len(products))
	for _, p := range products {
		out = append(out, toProto(p))
	}
	return &productv1.ListProductsResponse{Products: out}, nil
}

func (s *Server) UpdateStock(ctx context.Context, req *productv1.UpdateStockRequest) (*productv1.Product, error) {
	p, err := s.service.UpdateStock(ctx, req.GetProductId(), int(req.GetStock()))
	if err != nil {
		return nil, err
	}
	return toProto(p), nil
}

func (s *Server) ReserveStock(ctx context.Context, req *productv1.ReserveStockRequest) (*productv1.Product, error) {
	p, err := s.service.ReserveStock(ctx, req.GetProductId(), int(req.GetQuantity()))
	if err != nil {
		return nil, err
	}
	return toProto(p), nil
}

func (s *Server) ReleaseStock(ctx context.Context, req *productv1.ReleaseStockRequest) (*productv1.Product, error) {
	p, err := s.service.ReleaseStock(ctx, req.GetProductId(), int(req.GetQuantity()))
	if err != nil {
		return nil, err
	}
	return toProto(p), nil
}

func (s *Server) AddProductImage(ctx context.Context, req *productv1.AddProductImageRequest) (*productv1.ProductImage, error) {
	image, err := s.service.AddImage(ctx, product.ImageInput{
		ProductID:   req.GetProductId(),
		Filename:    req.GetFilename(),
		ContentType: req.GetContentType(),
		Content:     req.GetContent(),
	})
	if err != nil {
		return nil, err
	}
	return &productv1.ProductImage{
		Id:         image.ID,
		ProductId:  image.ProductID,
		ObjectName: image.Object,
		Url:        image.URL,
		CreatedAt:  formatTime(image.CreatedAt),
	}, nil
}

func toProto(p product.Product) *productv1.Product {
	return &productv1.Product{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		PriceKzt:    p.PriceKZT,
		Stock:       int32(p.Stock),
		ImageUrl:    p.ImageURL,
		CreatedAt:   formatTime(p.CreatedAt),
		UpdatedAt:   formatTime(p.UpdatedAt),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

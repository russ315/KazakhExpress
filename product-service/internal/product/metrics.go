package product

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ProductsCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kazakhexpress_products_created_total",
			Help: "Total number of created products",
		},
	)
	ProductStockUpdatesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_product_stock_updates_total",
			Help: "Total number of product stock updates",
		},
		[]string{"type"}, // update, reserve, release
	)
	ProductImageUploadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_product_image_uploads_total",
			Help: "Total number of image uploads",
		},
		[]string{"content_type"},
	)
	ProductDBOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_product_db_operations_total",
			Help: "Total number of product DB operations",
		},
		[]string{"operation", "status"}, // success, error
	)
	ProductDBOperationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kazakhexpress_product_db_operation_duration_seconds",
			Help:    "Duration of product DB operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(
		ProductsCreatedTotal,
		ProductStockUpdatesTotal,
		ProductImageUploadsTotal,
		ProductDBOperationsTotal,
		ProductDBOperationDurationSeconds,
	)
}

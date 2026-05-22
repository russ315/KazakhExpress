package order

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	OrdersCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kazakhexpress_orders_created_total",
			Help: "Total number of created orders",
		},
	)
	OrdersCancelledTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_orders_cancelled_total",
			Help: "Total number of cancelled orders",
		},
		[]string{"reason"},
	)
	OrderRevenueKZTTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kazakhexpress_order_revenue_kzt_total",
			Help: "Total revenue generated from orders in KZT",
		},
	)
	OrderStatusTransitionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_order_status_transitions_total",
			Help: "Total number of order status transitions",
		},
		[]string{"from_status", "to_status"},
	)
	OrderDBOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_order_db_operations_total",
			Help: "Total number of order DB operations",
		},
		[]string{"operation", "status"}, // success, error
	)
	OrderDBOperationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kazakhexpress_order_db_operation_duration_seconds",
			Help:    "Duration of order DB operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(
		OrdersCreatedTotal,
		OrdersCancelledTotal,
		OrderRevenueKZTTotal,
		OrderStatusTransitionsTotal,
		OrderDBOperationsTotal,
		OrderDBOperationDurationSeconds,
	)
}

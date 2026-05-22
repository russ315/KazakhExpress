package payment

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	PaymentAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_payment_attempts_total",
			Help: "Total number of payment attempts",
		},
		[]string{"method"},
	)
	PaymentSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_payment_success_total",
			Help: "Total number of successful payments",
		},
		[]string{"method"},
	)
	PaymentFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_payment_failures_total",
			Help: "Total number of failed payments",
		},
		[]string{"method", "reason"},
	)
	PaymentAmountKZTTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kazakhexpress_payment_amount_kzt_total",
			Help: "Total amount of processed payments in KZT",
		},
	)
	PaymentProcessingDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kazakhexpress_payment_processing_duration_seconds",
			Help:    "Duration of payment processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)
	PaymentDBOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_payment_db_operations_total",
			Help: "Total number of payment DB operations",
		},
		[]string{"operation", "status"}, // success, error
	)
	PaymentDBOperationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kazakhexpress_payment_db_operation_duration_seconds",
			Help:    "Duration of payment DB operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(
		PaymentAttemptsTotal,
		PaymentSuccessTotal,
		PaymentFailuresTotal,
		PaymentAmountKZTTotal,
		PaymentProcessingDurationSeconds,
		PaymentDBOperationsTotal,
		PaymentDBOperationDurationSeconds,
	)
}

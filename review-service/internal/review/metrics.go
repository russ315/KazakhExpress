package review

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ReviewsCreatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_reviews_created_total",
			Help: "Total number of reviews created",
		},
		[]string{"rating"},
	)
	ReviewsDeletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kazakhexpress_reviews_deleted_total",
			Help: "Total number of reviews deleted",
		},
	)
	ReviewRatingsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_review_ratings_total",
			Help: "Count of ratings left per star category",
		},
		[]string{"rating"},
	)
	ReviewDBOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_review_db_operations_total",
			Help: "Total number of review DB operations",
		},
		[]string{"operation", "status"}, // success, error
	)
	ReviewDBOperationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kazakhexpress_review_db_operation_duration_seconds",
			Help:    "Duration of review DB operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(
		ReviewsCreatedTotal,
		ReviewsDeletedTotal,
		ReviewRatingsTotal,
		ReviewDBOperationsTotal,
		ReviewDBOperationDurationSeconds,
	)
}

func RecordRatingLeft(rating int) {
	ratingStr := strconv.Itoa(rating)
	ReviewRatingsTotal.WithLabelValues(ratingStr).Inc()
}

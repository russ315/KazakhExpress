package user

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	UserRegistrationsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kazakhexpress_user_registrations_total",
			Help: "Total number of successful user registrations",
		},
	)
	UserLoginsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_user_logins_total",
			Help: "Total number of user login attempts",
		},
		[]string{"status"}, // success, failed, rate_limited
	)
	UserCacheRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_user_cache_requests_total",
			Help: "Total number of user profile cache requests",
		},
		[]string{"hit"}, // true, false
	)
	UserDBOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_user_db_operations_total",
			Help: "Total number of user DB operations",
		},
		[]string{"operation", "status"}, // success, error
	)
	UserDBOperationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kazakhexpress_user_db_operation_duration_seconds",
			Help:    "Duration of user DB operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	prometheus.MustRegister(
		UserRegistrationsTotal,
		UserLoginsTotal,
		UserCacheRequestsTotal,
		UserDBOperationsTotal,
		UserDBOperationDurationSeconds,
	)
}

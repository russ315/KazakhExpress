package smtp

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	EmailsSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_emails_sent_total",
			Help: "Total number of emails sent successfully",
		},
		[]string{"type"}, // welcome, receipt, refund, failure, unknown
	)
	EmailFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kazakhexpress_email_failures_total",
			Help: "Total number of email sending failures",
		},
		[]string{"type", "reason"},
	)
	EmailDeliveryDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kazakhexpress_email_delivery_duration_seconds",
			Help:    "Duration of email sending in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type", "provider"}, // provider: smtp, resend, dry_run
	)
)

func init() {
	prometheus.MustRegister(
		EmailsSentTotal,
		EmailFailuresTotal,
		EmailDeliveryDurationSeconds,
	)
}

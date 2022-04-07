package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/razorpay/trino-gateway/internal/boot"
)

var (
	RequestsReceivedTotal *prometheus.CounterVec
	ResponsesSentTotal    *prometheus.CounterVec
	ResponseDurations     *prometheus.HistogramVec
	FallbackGroupInvoked  *prometheus.CounterVec
)

func init() {
	env := boot.Config.App.Env
	RequestsReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_http_requests_total",
			Help: "Number of HTTP requests received.",
		},
		[]string{"env", "package", "server", "method"},
	).MustCurryWith(prometheus.Labels{"env": env})

	ResponsesSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_http_responses_total",
			Help: "Number of HTTP responses sent.",
		},
		[]string{"env", "package", "server", "method", "code"},
	).MustCurryWith(prometheus.Labels{"env": env})

	ResponseDurations = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trino_gateway_http_durations_ms_histogram",
			Help:    "HTTP latency distributions histogram.",
			Buckets: []float64{2, 5, 10, 15, 25, 40, 60, 85, 120, 150, 200, 300},
		},
		[]string{"env", "package", "server", "method", "code"},
	).MustCurryWith(prometheus.Labels{"env": env}).(*prometheus.HistogramVec)

	FallbackGroupInvoked = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_fallback_group_invoked_total",
			Help: "Number of requests where fallback group routing was invoked",
		},
		[]string{"env"},
	).MustCurryWith(prometheus.Labels{"env": env})
}

package router

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/razorpay/trino-gateway/internal/boot"
)

type Metrics struct {
	requestsReceivedTotal     *prometheus.CounterVec
	requestsRoutedTotal       *prometheus.CounterVec
	requestPreRoutingDelays   *prometheus.HistogramVec
	requestPostRoutingDelays  *prometheus.HistogramVec
	responsesSentTotal        *prometheus.CounterVec
	responseDurations         *prometheus.HistogramVec
	smartRouterDecisions      *prometheus.CounterVec
	smartRouterErrors         prometheus.Counter
	smartRouterInvalidBackend prometheus.Counter
}

var metrics *Metrics

func initMetrics() {
	env := boot.Config.App.Env
	metrics = &Metrics{}
	metrics.requestsReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_router_http_requests_total",
			Help: "Number of HTTP requests received from clients.",
		},
		[]string{"env", "method", "port"},
	).MustCurryWith(prometheus.Labels{"env": env})

	metrics.requestsRoutedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_router_http_requests_routed_total",
			Help: "Number of HTTP requests routed to a trino server.",
		},
		[]string{"env", "method", "port", "group", "backend"},
	).MustCurryWith(prometheus.Labels{"env": env})

	metrics.smartRouterDecisions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_router_smart_decisions_total",
			Help: "Decisions returned by the smart router, by mode, decision source, tier and reject.",
		},
		[]string{"env", "mode", "source", "tier", "reject"},
	).MustCurryWith(prometheus.Labels{"env": env})

	metrics.smartRouterErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_router_smart_errors_total",
			Help: "Smart router calls that failed or timed out; the strategy backend was kept.",
		},
		[]string{"env"},
	).WithLabelValues(env)

	metrics.smartRouterInvalidBackend = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_router_smart_invalid_backend_total",
			Help: "Enforce-mode decisions naming a backend outside the group or tiers, or not enabled and healthy; the strategy backend was kept.",
		},
		[]string{"env"},
	).WithLabelValues(env)

	metrics.requestPreRoutingDelays = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trino_gateway_router_http_pre_routing_delay_ms_histogram",
			Help:    "Delay in routing client request to a Trino server, latency distributions histogram.",
			Buckets: []float64{5, 10, 15, 20, 30, 40, 60, 100, 150, 500},
		},
		[]string{"env", "method"},
	).MustCurryWith(prometheus.Labels{"env": env}).(*prometheus.HistogramVec)

	metrics.requestPostRoutingDelays = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trino_gateway_router_http_post_routing_delay_ms_histogram",
			Help:    "Delay in sending response to client after receiving response from Trino server, latency distributions histogram.",
			Buckets: []float64{2, 5, 10, 15, 20, 25, 30, 40, 50, 100, 500},
		},
		[]string{"env", "method", "code"},
	).MustCurryWith(prometheus.Labels{"env": env}).(*prometheus.HistogramVec)

	metrics.responsesSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_router_http_responses_total",
			Help: "Number of HTTP responses sent back to client.",
		},
		[]string{"env", "method", "code"},
	).MustCurryWith(prometheus.Labels{"env": env})

	metrics.responseDurations = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trino_gateway_router_http_durations_ms_histogram",
			Help:    "Router HTTP latency distributions histogram for responses sent to clients.",
			Buckets: []float64{20, 40, 60, 90, 120, 150, 200, 250, 300, 500},
		},
		[]string{"env", "method", "code"},
	).MustCurryWith(prometheus.Labels{"env": env}).(*prometheus.HistogramVec)
}

package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/razorpay/trino-gateway/internal/boot"
)

type Metrics struct {
	executionsTotal    *prometheus.CounterVec
	executionlastRunAt *prometheus.GaugeVec
	executionDurations *prometheus.HistogramVec
	backendLoad        *prometheus.GaugeVec
}

var metrics *Metrics

func initMetrics() {
	env := boot.Config.App.Env
	metrics = &Metrics{}
	metrics.executionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trino_gateway_monitor_executions_total",
			Help: "Number of executions triggered for monitor task.",
		},
		[]string{"env"},
	).MustCurryWith(prometheus.Labels{"env": env})

	metrics.executionlastRunAt = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trino_gateway_monitor_execution_last_run_at",
			Help: "Monitor task last run epoch ts.",
		},
		[]string{"env"},
	).MustCurryWith(prometheus.Labels{"env": env})

	metrics.executionDurations = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trino_gateway_monitor_execution_seconds_histogram",
			Help:    "Monitor task execution time distributions histogram.",
			Buckets: []float64{5, 15, 30, 60, 90, 120, 150, 180, 210, 240},
		},
		[]string{"env"},
	).MustCurryWith(prometheus.Labels{"env": env}).(*prometheus.HistogramVec)

	metrics.backendLoad = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trino_gateway_monitor_backend_load",
			Help: "Backend Load computed by last run of monitor task.",
		},
		[]string{"env", "backend"},
	).MustCurryWith(prometheus.Labels{"env": env})
}

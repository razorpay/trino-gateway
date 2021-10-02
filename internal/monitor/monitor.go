package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/razorpay/trino-gateway/internal/provider"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

type Monitor struct {
	core ICore
}

func init() {
	initMetrics()
}

func NewMonitor(core ICore) *Monitor {
	return &Monitor{
		core: core,
	}
}

func (m *Monitor) Schedule(ctx *context.Context, interval string) error {
	s := gocron.NewScheduler(time.UTC)
	j, err := s.Every(interval).Do(m.Execute, ctx)

	if err != nil {
		return err
	}
	s.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
	s.StartImmediately().StartAsync()

	provider.Logger(*ctx).Infow("Scheduled Monitoring Job", map[string]interface{}{
		"job":        fmt.Sprintf("%v", j),
		"nextRunUTC": j.NextRun().Local().UTC(),
	})

	return nil
}

func (m *Monitor) Execute(ctx *context.Context) {
	provider.Logger(*ctx).Info("Executing monitoring task")

	metrics.executionsTotal.
		WithLabelValues(metrics.env).
		Inc()
	defer func(st time.Time) {
		duration := float64(time.Since(st).Seconds())
		metrics.executionDurations.
			WithLabelValues(metrics.env).
			Observe(duration)

		metrics.executionlastRunAt.
			WithLabelValues(metrics.env).
			SetToCurrentTime()
	}(time.Now())

	provider.Logger(*ctx).Info("Enabling/Disabling backends based on uptime schedules")
	err := m.core.EvaluateUptimeSchedules(ctx)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Error("Error evaluating backend uptime schedules")
	}

	provider.Logger(*ctx).Info("Enabling/Disabling backends based on cluster load")
	provider.Logger(*ctx).Debug("Fetching all currently active backends")
	active, err := m.core.GetActiveBackends(ctx)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Error("Unable to fetch all currently active backends")
		return
	}
	if len(active) == 0 {
		provider.Logger(*ctx).Error("No Active Backends detected")
	}
	var wg sync.WaitGroup
	for _, b := range active {
		provider.Logger(*ctx).Debugw(
			"Updating backend as per its health & clusterLoad",
			map[string]interface{}{"backend": b})

		wg.Add(1)
		go func(x *gatewayv1.Backend) {
			defer wg.Done()
			x = m.checkBackendStatus(ctx, x)
			err = m.core.UpdateBackend(ctx, x)
			if err != nil {
				provider.Logger(*ctx).WithError(err).Errorw(
					"Failure updating backend",
					map[string]interface{}{"backend": x})
			}
		}(b)

	}
	// Wait for all backend status checks to complete.
	wg.Wait()
}

func (m *Monitor) checkBackendStatus(ctx *context.Context, b *gatewayv1.Backend) *gatewayv1.Backend {
	isHealthy, err := m.core.IsBackendHealthy(ctx, b)
	if err != nil {
		b.IsEnabled = false
		provider.Logger(*ctx).WithError(err).Errorw(
			"Failure fetching backend health, marked as inactive",
			map[string]interface{}{"backend": b})
	} else {
		if isHealthy {
			load, err := m.core.GetBackendLoad(ctx, b)
			if err != nil {
				b.IsEnabled = false
				provider.Logger(*ctx).WithError(err).Errorw(
					"Failure evaluating current backend load, marked as inactive",
					map[string]interface{}{"backend": b})

			}
			b.ClusterLoad = load
			curr := time.Now().Unix()
			b.StatsUpdatedAt = curr

			if b.ThresholdClusterLoad == 0 || b.ClusterLoad <= b.ThresholdClusterLoad {
				b.IsEnabled = true
				provider.Logger(*ctx).Debugw(
					"Cluster load below threshold, marked as active",
					map[string]interface{}{"backend": b})
			}
		} else {
			b.IsEnabled = false
			provider.Logger(*ctx).Debugw(
				"Cluster unhealthy, marked as inactive",
				map[string]interface{}{"backend": b})
		}
	}
	return b
}

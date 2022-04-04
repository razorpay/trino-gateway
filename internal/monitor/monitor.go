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

	provider.Logger(*ctx).Info("Evaluating new state for backends")
	newSts, err := m.core.EvaluateBackendNewState(ctx)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Error("Error evaluating new states for backends")
		return
	}

	provider.Logger(*ctx).Debug("Deactivating backends")
	var wg sync.WaitGroup
	for _, b := range newSts.Inactive {
		wg.Add(1)
		go func(x *gatewayv1.Backend) {
			defer wg.Done()
			err = m.core.DeactivateBackend(ctx, x)
			if err != nil {
				provider.Logger(*ctx).WithError(err).Errorw(
					"Failure deactivating backend",
					map[string]interface{}{"backend": x})
			}
		}(b)
	}

	if len(newSts.Active) == 0 {
		provider.Logger(*ctx).Error("No Backends eligible for active state")
	}

	provider.Logger(*ctx).Info("Activating/Deactivating backends based on cluster load")
	for _, b := range newSts.Active {
		wg.Add(1)
		go func(x *gatewayv1.Backend) {
			defer wg.Done()
			provider.Logger(*ctx).Debugw(
				"Evaluating health of backend",
				map[string]interface{}{"backend": b})

			isHealthy, err := m.core.IsBackendHealthy(ctx, x)
			if err != nil {
				provider.Logger(*ctx).WithError(err).Errorw(
					"Failure checking backend health",
					map[string]interface{}{"backend": x})
			}

			if isHealthy {
				err := m.core.ActivateBackend(ctx, x)
				if err != nil {
					provider.Logger(*ctx).WithError(err).Errorw(
						"Failure activating backend",
						map[string]interface{}{"backend": x})
				}
			} else {

				err = m.core.DeactivateBackend(ctx, x)
				if err != nil {
					provider.Logger(*ctx).WithError(err).Errorw(
						"Failure deactivating backend",
						map[string]interface{}{"backend": x})
				}
			}
		}(b)

	}

	// Wait for all backend status checks to complete.
	wg.Wait()
}

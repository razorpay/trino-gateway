package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/razorpay/trino-gateway/internal/provider"
	gatewayv1 "github.com/razorpay/trino-gateway/rpc/gateway"
)

type Monitor struct {
	core      ICore
	scheduler gocron.Scheduler
}

func init() {
	initMetrics()
}

func NewMonitor(core ICore) (*Monitor, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	return &Monitor{
		core:      core,
		scheduler: s,
	}, nil
}

func (m *Monitor) Schedule(ctx *context.Context, interval string) error {
	parsedInterval, err := time.ParseDuration(interval)
	if err != nil {
		return err
	}
	j, err := m.scheduler.NewJob(
		gocron.DurationJob(parsedInterval),
		gocron.NewTask(m.Execute, ctx),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		return err
	}
	// m.scheduler.StartAsync()

	nextRunAt, _ := j.NextRun()
	provider.Logger(*ctx).Infow("Scheduled Monitoring Job", map[string]interface{}{
		"job":        fmt.Sprintf("%v", j),
		"nextRunUTC": nextRunAt.Local().UTC(),
	})

	return nil
}

func (m *Monitor) Teardown(ctx *context.Context) error {
	// when you're done, shut it down
	err := m.scheduler.Shutdown()
	return err
}

func (m *Monitor) Execute(ctx *context.Context) {
	provider.Logger(*ctx).Info("Executing monitoring task")

	metrics.executionsTotal.
		WithLabelValues().Inc()

	defer func(st time.Time) {
		duration := float64(time.Since(st).Seconds())
		metrics.executionDurations.
			WithLabelValues().Observe(duration)

		metrics.executionlastRunAt.
			WithLabelValues().SetToCurrentTime()
	}(time.Now())

	provider.Logger(*ctx).Info("Evaluating new state for backends")
	newStates, err := m.core.EvaluateBackendNewState(ctx)
	if err != nil {
		provider.Logger(*ctx).WithError(err).Error("Error evaluating new states for backends")
		return
	}

	if len(newStates.Healthy) == 0 {
		provider.Logger(*ctx).Error("No Backends are in Healthy state.")
	}

	provider.Logger(*ctx).Debug("Marking healthy/unhealthy backends as per evaluated states")
	var wg sync.WaitGroup
	for _, b := range newStates.Unhealthy {
		wg.Add(1)
		go func(x *gatewayv1.Backend) {
			defer wg.Done()
			err = m.core.MarkUnhealthyBackend(ctx, x)
			if err != nil {
				provider.Logger(*ctx).WithError(err).Errorw(
					"Failure marking backend as Unhealthy",
					map[string]interface{}{"backend": x})
			}
		}(b)
	}

	for _, b := range newStates.Healthy {
		wg.Add(1)
		go func(x *gatewayv1.Backend) {
			defer wg.Done()
			err = m.core.MarkHealthyBackend(ctx, x)
			if err != nil {
				provider.Logger(*ctx).WithError(err).Errorw(
					"Failure marking backend as Healthy",
					map[string]interface{}{"backend": x})
			}
		}(b)
	}

	// Wait for all backend health updates to complete.
	wg.Wait()

	provider.Logger(*ctx).Info("Finished executing monitoring task")
}

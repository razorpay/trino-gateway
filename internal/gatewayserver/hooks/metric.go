package hooks

import (
	"context"
	"fmt"
	"time"

	"github.com/razorpay/trino-gateway/internal/gatewayserver/metrics"
	"github.com/twitchtv/twirp"
)

var reqStartTsCtxKey = new(int)

// Metric returns function which puts unique request id into context.
func Metric() *twirp.ServerHooks {
	hooks := &twirp.ServerHooks{}

	// RequestReceived:
	hooks.RequestReceived = func(ctx context.Context) (context.Context, error) {
		ctx = markRequestStart(ctx)

		return ctx, nil
	}

	// RequestRouted:
	hooks.RequestRouted = func(ctx context.Context) (context.Context, error) {
		pkg, _ := twirp.PackageName(ctx)
		service, _ := twirp.ServiceName(ctx)
		method, _ := twirp.MethodName(ctx)

		metrics.RequestsReceivedTotal.
			WithLabelValues(pkg, service, method).
			Inc()

		return ctx, nil
	}

	// ResponseSent:
	hooks.ResponseSent = func(ctx context.Context) {
		start, _ := getRequestStart(ctx)
		pkg, _ := twirp.PackageName(ctx)
		service, _ := twirp.ServiceName(ctx)
		method, _ := twirp.MethodName(ctx)
		statusCode, _ := twirp.StatusCode(ctx)

		duration := float64(time.Now().Sub(start).Milliseconds())

		metrics.ResponsesSentTotal.WithLabelValues(
			pkg, service, method,
			fmt.Sprintf("%v", statusCode),
		).Inc()

		metrics.ResponseDurations.WithLabelValues(
			pkg, service, method,
			fmt.Sprintf("%v", statusCode),
		).Observe(duration)
	}

	return hooks
}

func markRequestStart(ctx context.Context) context.Context {
	return context.WithValue(ctx, reqStartTsCtxKey, time.Now())
}

func getRequestStart(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(reqStartTsCtxKey).(time.Time)
	return t, ok
}

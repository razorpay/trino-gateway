package provider

import (
	"context"

	"github.com/razorpay/trino-gateway/pkg/logger"
)

// Logger will provider the logger instance
func Logger(ctx context.Context) *logger.Entry {
	ctxLogger, err := logger.Ctx(ctx)

	if err == nil {
		return ctxLogger
	}

	return nil
}

package worker_test

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/razorpay/trino-gateway/pkg/worker"
)

func TestConfig_SetDefaults(t *testing.T) {
	c := &worker.Config{}

	c.SetDefaults()

	assert.Equal(t, c, &worker.Config{
		Name:           worker.DefaultName,
		MaxConcurrency: worker.DefaultConcurrency,
		WaitTime:       worker.DefaultWaitTime,
		RetryDelay:     worker.DefaultRetryDelay,
	})
}

package util

import (
	"context"
	"testing"
	"time"

	"github.com/razorpay/trino-gateway/pkg/logger"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type CoreSuite struct {
	suite.Suite
	ctx *context.Context
}

func (suite *CoreSuite) SetupTest() {
	lgrConfig := logger.Config{
		LogLevel: logger.Warn,
	}

	l, err := logger.NewLogger(lgrConfig)
	if err != nil {
		panic("failed to initialize logger")
	}

	c := context.WithValue(context.Background(), logger.LoggerCtxKey, l)

	suite.ctx = &c
}

func (suite *CoreSuite) Test_IsTimeInCron() {
	// func (c *Core) isCurrentTimeInCron(ctx *context.Context, sched string) (bool, error)

	tst := func(t time.Time, sched string) bool {
		s, _ := IsTimeInCron(suite.ctx, t, sched)
		return s
	}

	suite.Equalf(
		true,
		tst(time.Now(), "* * * * *"),
		"Failure",
	)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(CoreSuite))
}

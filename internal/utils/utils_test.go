package utils

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/razorpay/trino-gateway/pkg/logger"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type UtilsSuite struct {
	suite.Suite
	ctx *context.Context
}

func (suite *UtilsSuite) SetupTest() {
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

func (suite *UtilsSuite) Test_IsTimeInCron() {
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

func (suite *UtilsSuite) Test_stringifyHttpRequest() {
}

func (suite *UtilsSuite) Test_stringifyHttpResponse() {
}

func (suite *UtilsSuite) Test_parseBody() {
	str := "body"
	stringReader := strings.NewReader(str)
	stringReadCloser := io.NopCloser(stringReader)

	tst := func() string {
		s, _ := ParseHttpPayloadBody(suite.ctx, &stringReadCloser)
		return s
	}

	suite.Equalf(str, tst(), "Failed to extract string from body")
	suite.Equalf(str, tst(), "String extraction is not idempotent")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}

package router

import (
	"context"
	"testing"

	"github.com/razorpay/trino-gateway/pkg/logger"
	"github.com/stretchr/testify/suite"
)

type HelpersSuite struct {
	suite.Suite
	ctx *context.Context
}

func (suite *HelpersSuite) SetupTest() {
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

func (suite *HelpersSuite) Test_extractQueryId() {
}

func (suite *HelpersSuite) Test_isValidRequest() {
}

func (suite *HelpersSuite) Test_constructQueryFromReq() {
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(HelpersSuite))
}

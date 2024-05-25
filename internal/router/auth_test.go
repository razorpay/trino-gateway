package router

import (
	"context"
	"testing"

	"github.com/razorpay/trino-gateway/pkg/logger"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type AuthSuite struct {
	suite.Suite
	authService *AuthService
	ctx         *context.Context
}

func (suite *AuthSuite) SetupTest() {
	lgrConfig := logger.Config{
		LogLevel: logger.Warn,
	}

	l, err := logger.NewLogger(lgrConfig)
	if err != nil {
		panic("failed to initialize logger")
	}

	c := context.WithValue(context.Background(), logger.LoggerCtxKey, l)

	suite.ctx = &c
	suite.authService = &AuthService{}
}

func (suite *AuthSuite) Test_GetInMemoryAuthCache_Persistance() {
	key := "testKey"
	value := "testValue"

	authCache := suite.authService.GetInMemoryAuthCache(suite.ctx)
	authCache.Update(key, value)

	authCacheInstance2 := suite.authService.GetInMemoryAuthCache(suite.ctx)
	entry, exists := authCacheInstance2.Get(key)

	suite.Truef(exists, "Second cache instance doesn't have same key")
	if exists {
		suite.Equalf(value, entry, "Second Cache instance value doesn't match.")
	}

}
func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthSuite))
}

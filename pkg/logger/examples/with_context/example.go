package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/razorpay/trino-gateway/pkg/logger"
)

func main() {
	// Default Config with sentry disabled
	config := logger.Config{
		LogLevel:       logger.Debug,
		SentryDSN:      "",
		SentryEnabled:  false,
		SentryLogLevel: "",
	}
	lgr, err := logger.NewLogger(config)
	if err != nil {
		fmt.Printf("Error getting logger:%v", err)
		os.Exit(1)
	}

	// Fist, simply log
	lgr.Debug("Simply log here and do nothing")

	// Add default fields for this logger that needs to be attached to very log message
	basicLogger := lgr.WithFields(map[string]interface{}{"key1": "value1"})

	// default logging with no extra fields
	basicLogger.Debug("I have the default values for Key1 and Value1")

	// add additional custom fields
	basicLogger.Debugw("I have default values and additional extra fields", map[string]interface{}{"key2": "value2"})

	// Now create a logger based on certain context fields from the existing logger
	c := context.WithValue(context.Background(), "myCtxKey", "myCtxValue")
	ctxLogger := basicLogger.WithContext(c, []string{"myCtxKey"})
	ctxLogger.Debug("Default logging with Context")

	// Now, set the above logger to context, retrieve and log again
	c = context.Background()
	ctx := context.WithValue(c, logger.LoggerCtxKey, ctxLogger)
	fromContextLogger, err := logger.Ctx(ctx)
	if err != nil {
		fmt.Printf("Error obtaining logger from context")
		os.Exit(1)
	}
	fromContextLogger.Debug("Logging obtained from existing context")

	// Field Chaining
	basicLogger.WithFields(map[string]interface{}{"foo": "bar"}).Debug("Another message")

	//With Custom error
	basicLogger.WithError(errors.New("Doomed here")).Debug("Error message")
}

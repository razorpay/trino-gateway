package main

import (
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

}

package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/razorpay/trino-gateway/internal/boot"
	trino_rest "github.com/razorpay/trino-gateway/internal/trino_rest"
)

func main() {
	//Initialize context
	ctx, cancel := context.WithCancel(boot.NewContext(context.Background()))
	defer cancel()

	boot.TrinoRestInit()
	logger := boot.InitLoggerTrinoRest(ctx)

	app, err := trino_rest.NewApp(&boot.Config)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to initialize app: %v", err))
	}

	logger.Info(fmt.Sprintf("Starting server on: %d", boot.Config.App.Port))
	logger.Fatal(fmt.Sprint(http.ListenAndServe(":"+strconv.Itoa(boot.Config.App.Port), app.Router)))
}

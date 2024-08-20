package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"trino-api/internal/app"
	"trino-api/internal/boot"
)

func main() {
	//Initialize context
	ctx, cancel := context.WithCancel(boot.NewContext(context.Background()))
	defer cancel()

	boot.Init()
	logger := boot.InitLogger(ctx)

	app, err := app.NewApp(&boot.Config)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to initialize app: %v", err))
	}
	defer app.Close()

	logger.Info(fmt.Sprintf("Starting server on: %d", boot.Config.App.Port))
	logger.Fatal(fmt.Sprint(http.ListenAndServe(":"+strconv.Itoa(boot.Config.App.Port), app.Router)))
}

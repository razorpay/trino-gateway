package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"trino-api/internal/app"
	"trino-api/internal/boot"

	log "github.com/sirupsen/logrus"
)

func main() {
	//Initialize context
	ctx, cancel := context.WithCancel(boot.NewContext(context.Background()))
	defer cancel()

	// env := boot.GetEnv()
	// err := boot.InitApi(ctx, env)
	// if err != nil {
	// 	log.Fatalf("failed to inti api: %v", err)
	// }
	// boot.InitTracing()
	boot.Init()
	// logger := boot.InitLogger(ctx)
	// logger := log
	log.Debug(ctx)

	app, err := app.NewApp(boot.Config.Db.DSN)
	if err != nil {
		// logger.Fatal(fmt.Sprintf("Failed to initialize app: %v", err))
		log.Fatalf(fmt.Sprintf("Failed to initialize app: %v", err))
	}
	defer app.Close()

	log.Info(fmt.Sprintf("Starting server on: %d", boot.Config.App.Port))
	log.Fatal(fmt.Sprint(http.ListenAndServe(":"+strconv.Itoa(boot.Config.App.Port), app.Router)))
}

package app

import (
	"fmt"
	"trino-api/internal/app/handler"
	"trino-api/internal/app/routes"
	"trino-api/internal/config"
	"trino-api/internal/services/trino"

	"github.com/gorilla/mux"
)

type App struct {
	Router      *mux.Router
	TrinoClient *trino.Client
	Handler     *handler.Handler
}

func NewApp(cfg *config.Config) (*App, error) {

	if cfg.Db.DSN == "" {
		return nil, fmt.Errorf("configuration or database settings are missing")
	}

	trinoClient, err := trino.NewTrinoClient(cfg.Db.DSN)
	if err != nil {
		return nil, err
	}

	handler := *handler.NewHandler(trinoClient, cfg)

	router := mux.NewRouter()
	routes.RegisterRoutes(router, &handler)

	return &App{
		Router:      router,
		TrinoClient: trinoClient,
		Handler:     &handler,
	}, nil
}

func (app *App) Close() error {
	return app.TrinoClient.Close()
}

package app

import (
	"fmt"
	"trino-api/internal/app/handler"
	"trino-api/internal/app/routes"
	"trino-api/internal/services/trino"

	"github.com/gorilla/mux"
)

type App struct {
	Router      *mux.Router
	TrinoClient *trino.Client
	Handler     *handler.Handler
}

func NewApp(dsn string) (*App, error) {

	if dsn == "" {
		return nil, fmt.Errorf("configuration or database settings are missing")
	}

	trinoClient, err := trino.NewTrinoClient(dsn)
	if err != nil {
		return nil, err
	}

	handler := *handler.NewHandler(trinoClient)

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

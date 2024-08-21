package trino_rest

import (
	"fmt"

	"github.com/razorpay/trino-gateway/internal/config"
	"github.com/razorpay/trino-gateway/internal/trino_rest/handler"
	"github.com/razorpay/trino-gateway/internal/trino_rest/routes"
	"github.com/razorpay/trino-gateway/internal/trino_rest/services/trino"

	"github.com/gorilla/mux"
)

type App struct {
	Router      *mux.Router
	TrinoClient *trino.Client
	Handler     *handler.Handler
}

func NewApp(cfg *config.Config) (*App, error) {

	if cfg.TrinoRest.TrinoBackendDB.DSN == "" {
		return nil, fmt.Errorf("configuration or database settings are missing")
	}

	trinoClient, err := trino.NewTrinoClient(cfg.TrinoRest.TrinoBackendDB.DSN)
	if err != nil {
		return nil, err
	}

	handler := *handler.NewHandler(trinoClient, cfg, nil)

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

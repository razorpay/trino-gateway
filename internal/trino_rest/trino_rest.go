package trino_rest

import (
	"github.com/razorpay/trino-gateway/internal/config"
	"github.com/razorpay/trino-gateway/internal/router"
	"github.com/razorpay/trino-gateway/internal/trino_rest/handler"
	"github.com/razorpay/trino-gateway/internal/trino_rest/routes"

	"github.com/gorilla/mux"
)

type App struct {
	Router      *mux.Router
	Handler     *handler.Handler
	AuthService *router.AuthService
}

func NewApp(cfg *config.Config) (*App, error) {
	authService := &router.AuthService{}

	handler := *handler.NewHandler(cfg, nil)

	router := mux.NewRouter()
	routes.RegisterRoutes(router, &handler, authService)

	return &App{
		Router:      router,
		Handler:     &handler,
		AuthService: authService,
	}, nil
}

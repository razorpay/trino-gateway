package routes

import (
	"github.com/razorpay/trino-gateway/internal/boot"
	"github.com/razorpay/trino-gateway/internal/router"
	"github.com/razorpay/trino-gateway/internal/trino_rest/handler"
	"github.com/razorpay/trino-gateway/internal/trino_rest/middleware"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, h *handler.Handler, authService *router.AuthService) {
	authMiddleware := middleware.NewAuthMiddleware(authService, &boot.Config)
	router.Use(authMiddleware.BasicAuthenticator)
	router.HandleFunc("/v1/query", h.QueryHandler()).Methods("POST")
}

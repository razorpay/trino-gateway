package routes

import (
	"trino-api/internal/app/handler"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, h *handler.Handler) {
	router.HandleFunc("/v1/query", h.QueryHandler()).Methods("POST")
	router.HandleFunc("/v1/health", h.HealthCheck()).Methods("GET")
}

package routes

import (
	"github.com/razorpay/trino-gateway/internal/trino_rest/handler"

	"github.com/gorilla/mux"
)

func RegisterRoutes(router *mux.Router, h *handler.Handler) {
	router.HandleFunc("/v1/query", h.QueryHandler()).Methods("POST")
}

package utils

import (
	"encoding/json"
	"net/http"

	"github.com/razorpay/trino-gateway/internal/trino_rest/model"
)

// send back the response in the json format
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// send back the response with error in json format
func RespondWithError(w http.ResponseWriter, code int, message string) {
	resp := model.RespData{
		Status: "Error",
		Error: &model.Error{
			Message:   message,
			ErrorCode: int64(code),
		},
	}
	RespondWithJSON(w, code, resp)
}

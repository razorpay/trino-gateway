package utils

import (
	"encoding/json"
	"net/http"
	"trino-api/internal/model"
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
		Error: model.Error{
			Message:   message,
			ErrorCode: int64(code),
		},
	}
	RespondWithJSON(w, code, resp)
}

func RespondWithRunningStatus(w http.ResponseWriter, code int, message string) {
	resp := model.RespData{
		Status: "Running",
	}
	RespondWithJSON(w, code, resp)
}

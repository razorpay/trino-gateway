package handler

import (
	"encoding/json"
	"net/http"
	"trino-api/internal/app/process"
	"trino-api/internal/model"
	"trino-api/internal/services/trino"
	"trino-api/internal/utils"
)

type Handler struct {
	TrinoClient *trino.Client
}

// NewHandler initializes the Handler with the TrinoClient.
func NewHandler(trinoClient *trino.Client) *Handler {
	return &Handler{
		TrinoClient: trinoClient,
	}
}
func (h *Handler) QueryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req *model.ReqData
			err error
			// resp *model.RespData
		)
		// move this to middleware
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		rows, err := h.TrinoClient.Query(req.SQL)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Unable to query trino: "+err.Error())
			return
		}
		defer rows.Close()
		columns, rowData, err := process.QueryResult(rows)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnprocessableEntity, "Unable to process: "+err.Error())
			return
		}
		if len(rowData) > 5000 {
			utils.RespondWithError(w, http.StatusRequestEntityTooLarge, "Response data is too big")
			return
		}
		utils.RespondWithJSON(w, http.StatusAccepted, model.RespData{
			Status:  "Success",
			Columns: columns,
			Data:    rowData,
		})
	}
}

func (h *Handler) HealthCheck() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{"status": "OK"}
		utils.RespondWithJSON(w, http.StatusOK, response)
	}
}

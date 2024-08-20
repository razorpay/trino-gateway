package handler

import (
	"encoding/json"
	"net/http"
	"trino-api/internal/app/process"
	"trino-api/internal/config"
	"trino-api/internal/model"
	"trino-api/internal/services/trino"
	"trino-api/internal/utils"
)

type Handler struct {
	TrinoClient *trino.Client
	cfg         *config.Config
}

// NewHandler initializes the Handler with the TrinoClient and config.
func NewHandler(trinoClient *trino.Client, cfg *config.Config) *Handler {
	return &Handler{
		TrinoClient: trinoClient,
		cfg:         cfg,
	}
}
func (h *Handler) QueryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			req *model.ReqData
			err error
		)

		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		rows, err := h.TrinoClient.Query(req.SQL)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()
		columns, rowData, err := process.QueryResult(rows)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnprocessableEntity, "Unable to process: "+err.Error())
			return
		}
		if len(rowData) > h.cfg.App.MaxRecords {
			utils.RespondWithError(w, http.StatusRequestEntityTooLarge, "Response data is too big")
			return
		}
		utils.RespondWithJSON(w, http.StatusAccepted, model.RespData{
			Status:  "Success",
			Columns: columns,
			Data:    rowData,
			Error:   nil,
		})
	}
}

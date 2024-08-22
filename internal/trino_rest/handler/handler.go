package handler

import (
	"encoding/json"
	"net/http"

	"github.com/razorpay/trino-gateway/internal/config"
	"github.com/razorpay/trino-gateway/internal/trino_rest/model"
	"github.com/razorpay/trino-gateway/internal/trino_rest/process"
	"github.com/razorpay/trino-gateway/internal/trino_rest/services/trino"
	"github.com/razorpay/trino-gateway/internal/trino_rest/utils"
)

type Handler struct {
	TrinoClient    trino.TrinoClient
	cfg            *config.Config
	queryProcessor process.QueryProcessor
}

// NewHandler initializes the Handler with the TrinoClient and config.
func NewHandler(trinoClient trino.TrinoClient, cfg *config.Config, processor process.QueryProcessor) *Handler {
	if processor == nil {
		processor = &process.DefaultProcessor{}
	}
	return &Handler{
		TrinoClient:    trinoClient,
		cfg:            cfg,
		queryProcessor: processor,
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
		columns, rowData, err := h.queryProcessor.QueryResult(rows)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnprocessableEntity, "Unable to process: "+err.Error())
			return
		}
		if len(rowData) > h.cfg.TrinoRest.MaxRecords {
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

package model

type ReqData struct {
	SQL string `json:"sql"`
}

type RespData struct {
	Status  string                   `json:"status,omitempty"`
	Columns []Column                 `json:"columns,omitempty"`
	Data    []map[string]interface{} `json:"data,omitempty"`
	Error   *Error                   `json:"error,omitempty"`
}

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
type Error struct {
	Message   string `json:"message"`
	ErrorCode int64  `json:"errorCode"`
}

package model

type ReqData struct {
	SQL string `json:"sql"`
}

type RespData struct {
	Status  string    `json:"status,omitempty"`
	Columns []Column  `json:"columns,omitempty"`
	Data    [][]Datum `json:"data,omitempty"`
	Error   Error     `json:"error,omitempty"`
}

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Datum struct {
	Data interface{} `json:"data"`
}
type Error struct {
	Message   string `json:"message"`
	ErrorCode int64  `json:"errorCode"`
	ErrorName string `json:"errorName"`
	ErrorType string `json:"errorType"`
}

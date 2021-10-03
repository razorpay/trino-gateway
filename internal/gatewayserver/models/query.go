package models

import "github.com/razorpay/trino-gateway/pkg/spine"

// query model struct definition
type Query struct {
	spine.Model
	Text        string `json:"text"`
	ClientIp    string `json:"client_ip"`
	GroupId     string `json:"group_id"`
	BackendId   string `json:"backend_id"`
	Username    string `json:"username"`
	SubmittedAt int64  `json:"submitted_at"`
	ServerHost  string `json:"server_host"`
}

func (u *Query) TableName() string {
	return "queries"
}

func (u *Query) EntityName() string {
	return "query"
}

func (u *Query) SetDefaults() error {
	return nil
}

func (u *Query) Validate() error {
	return nil
}

package models

import "github.com/razorpay/trino-gateway/pkg/spine"

// policy model struct definition
type Policy struct {
	spine.Model
	RuleType        string  `json:"rule_type"`
	RuleValue       string  `json:"rule_value"`
	GroupId         string  `json:"group_id"`
	FallbackGroupId *string `json:"fallback_group_id"`
	IsEnabled       *bool   `json:"is_enabled" sql:"DEFAULT:true"`
	IsAuthDelegated *bool   `json:"is_auth_delegated" sql:"DEFAULT:false"`
}

func (u *Policy) TableName() string {
	return "policies"
}

func (u *Policy) EntityName() string {
	return "policy"
}

func (u *Policy) SetDefaults() error {
	return nil
}

func (u *Policy) Validate() error {
	return nil
}

package models

import "github.com/razorpay/trino-gateway/pkg/spine"

// group model struct definition
type Group struct {
	spine.Model
	Strategy              string `json:"strategy"`
	IsEnabled             bool   `json:"is_enabled" sql:"DEFAULT:true"`
	GroupBackendsMappings []GroupBackendsMapping
}

func (u *Group) TableName() string {
	return "groups_"
}

func (u *Group) EntityName() string {
	return "group"
}

func (u *Group) SetDefaults() error {
	return nil
}

func (u *Group) Validate() error {
	return nil
}

type GroupBackendsMapping struct {
	spine.Model
	GroupId   string `json:"group_id"`
	BackendId string `json:"backend_id"`
}

func (u *GroupBackendsMapping) TableName() string {
	return "groups_backend_mappings"
}

func (u *GroupBackendsMapping) EntityName() string {
	return "groups_backend_mapping"
}

func (u *GroupBackendsMapping) SetDefaults() error {
	return nil
}

func (u *GroupBackendsMapping) Validate() error {
	return nil
}

package models

import "github.com/razorpay/trino-gateway/pkg/spine"

// group model struct definition
type Group struct {
	spine.Model
	Strategy              *string                `json:"strategy"`
	IsEnabled             *bool                  `json:"is_enabled" sql:"DEFAULT:true"`
	LastRoutedBackend     *string                `json:"last_routed_backend"`
	GroupBackendsMappings []GroupBackendsMapping `gorm:"foreignKey:GroupId;references:ID"`
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
	ID        *int32 `json:"id" sql:"DEFAULT:NULL"`
	GroupId   string `json:"group_id" gorm:"primaryKey"`
	BackendId string `json:"backend_id" gorm:"primaryKey"`
}

func (u *GroupBackendsMapping) TableName() string {
	return "group_backends_mappings"
}

func (u *GroupBackendsMapping) EntityName() string {
	return "group_backends_mappings"
}

func (u *GroupBackendsMapping) SetDefaults() error {
	return nil
}

func (u *GroupBackendsMapping) Validate() error {
	return nil
}

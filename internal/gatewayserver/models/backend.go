package models

import (
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/razorpay/trino-gateway/pkg/spine"
)

// backend model struct definition
type Backend struct {
	spine.Model
	Hostname       string `json:"hostname"`
	Scheme         string `json:"scheme"`
	ExternalUrl    string `json:"external_url"`
	IsEnabled      bool   `json:"is_enabled"`
	UptimeSchedule string `json:"uptime_schedule"`
}

func (u *Backend) TableName() string {
	return "backends"
}

func (u *Backend) EntityName() string {
	return "backend"
}

func (u *Backend) SetDefaults() error {
	return nil
}

func (u *Backend) Validate() error {
	err := validation.ValidateStruct(u,
		// id, required, length non zero
		validation.Field(&u.ID, validation.Required, validation.RuneLength(1, 50)),

		// url, required, string, length 1-30
		validation.Field(&u.Hostname, validation.Required, validation.RuneLength(1, 50)),

		// Scheme, required, string,  Union(http, https)
		validation.Field(&u.Scheme, validation.Required, validation.In("http", "https")),

		// first_name, required, string, length 1-30
		validation.Field(&u.ExternalUrl, validation.Required, validation.RuneLength(1, 50)),

		// status, required, string
		// validation.Field(&u.IsEnabled, validation.Required, validation.In(true, false)),
	)

	return err
}

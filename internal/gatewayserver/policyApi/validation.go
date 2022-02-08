package policyapi

// import (
// 	validation "github.com/go-ozzo/ozzo-validation/v4"
// )

// func (cp *CreateParams) Validate() error {
// 	err := validation.ValidateStruct(cp,
// 		// id, required, length non zero
// 		validation.Field(&cp.ID, validation.Required, validation.RuneLength(1, 50)),

// 		// Hostname, required, string, length 1-50
// 		validation.Field(&cp.Hostname, validation.Required, validation.RuneLength(1, 50)),

// 		// Scheme, required, string,  Union(http, https)
// 		validation.Field(&cp.Scheme, validation.Required, validation.In("http", "https")),

// 		// // last_name, required, string, length 1-30
// 		// validation.Field(&cp.LastName, validation.Required, validation.RuneLength(1, 30)),
// 	)

// 	return err
// 	// if err == nil {
// 	// 	return nil
// 	// }

// 	// publicErr := errorclass.ErrValidationFailure.New("").
// 	// 	Wrap(err).
// 	// 	WithPublic(&errors.Public{
// 	// 		Description: err.Error(),
// 	// 	})

// 	// return publicErr
// }

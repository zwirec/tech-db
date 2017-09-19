package models

import "github.com/go-ozzo/ozzo-validation"

type Vote struct {
	Nickname string
	Voice    int32
}

func (v *Vote) Validate() error {
	return validation.ValidateStruct(v,
		validation.Field(v.Nickname, validation.Required),
		validation.Field(v.Voice, validation.Required, validation.In(1, -1)),
	)
}

package models

import (
	"time"
	"github.com/go-ozzo/ozzo-validation"
)

type Post struct {
	ID       int64
	Author   string
	Created  time.Time
	Forum    string
	IsEdited bool
	Message  string
	Parent   int64
	Thread   int32
}

func (p *Post) Validate() error {
	return validation.ValidateStruct(p,
		validation.Field(p.Author, validation.Required),
		validation.Field(p.Message, validation.Required),
	)
}

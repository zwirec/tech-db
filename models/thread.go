package models

import (
	"time"
	"github.com/go-ozzo/ozzo-validation"
	"regexp"
)

var threadRuleSlug = []validation.Rule{
	validation.Required,
	validation.Match(regexp.MustCompile(`^(\d|\w|-|_)*(\w|-|_)(\d|\w|-|_)*$`)),
}


type Thread struct {
	ID      int32
	Author  string
	Forum   string
	Message string
	Slug    string
	Title   string
	Votes   int32
	Created time.Time
}

func (t *Thread) Validate() error {
	return validation.ValidateStruct(t,
		validation.Field(t.Message, validation.Required),
		validation.Field(t.Author, validation.Required),
		validation.Field(t.Title, validation.Required),
		validation.Field(t.Slug, threadRuleSlug...),
	)
}

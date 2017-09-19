package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"regexp"
)

var regexpForumSlug = regexp.MustCompile(`^(\d|\w|-|_)*(\w|-|_)(\d|\w|-|_)*$`)

var forumRuleSlug = []validation.Rule{
	validation.Required,
	validation.Match(regexpForumSlug),
}



type Forum struct {
	Posts int64
	Slug string
	Threads int32
	Title string
	User string
}

func (f *Forum) Validate() error {
	return validation.ValidateStruct(f,
		validation.Field(f.Slug, forumRuleSlug...),
		validation.Field(f.Title, validation.Required),
		validation.Field(f.User, validation.Required),
	)
}
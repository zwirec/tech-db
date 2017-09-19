package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"regexp"
)


var regexpUsername = regexp.MustCompile(`[a-zA-Z0-9_.]*`)

var forumUsernameSlug = []validation.Rule{
	validation.Required,
	validation.Match(regexpUsername),
}

type User struct {
	Nickname string
	Fullname string
	Email string
	About string
}


func (u *User) Validate() error {
	return validation.ValidateStruct(u,
			validation.Field(&u.Nickname, forumUsernameSlug...),
			validation.Field(&u.Email, validation.Required, is.Email),
			validation.Field(&u.Fullname, validation.Required),
	)
}


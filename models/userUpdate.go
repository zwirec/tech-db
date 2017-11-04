package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/db"
)

//easyjson:json
type UserUpdate struct {
	About    *string
	Email    *string
	Fullname *string
}

func (uupd *UserUpdate) Validate() error {
	return validation.ValidateStruct(uupd,
		validation.Field(&uupd.Email, validation.Required, is.Email),
		validation.Field(&uupd.Fullname, validation.Required),
	)
}

func (u *UserUpdate) UpdateProfile(ctx *routing.Context) (*User, *Error) {
	tx, _ := database.DB.Begin()
	user := User{}
	if err := tx.QueryRow(`SELECT id FROM "user" u
									WHERE u.nickname = $1`,
		ctx.Param("nickname")).Scan(new(int)); err != nil {
		//1
		tx.Rollback()
		return nil, &Error{Type: ErrorNotFound}

	}
	////1
	if err := tx.QueryRow(`UPDATE "user" SET fullname = COALESCE($1, fullname),
													email = COALESCE($2, email),
													about = COALESCE($3, about)
									WHERE nickname = $4 RETURNING nickname::text, fullname, email::text, about`,
		u.Fullname, u.Email, u.About, ctx.Param("nickname")).
		Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About);
		err != nil {
		//1
		tx.Rollback()
		return nil, &Error{Type: ErrorAlreadyExists}
	}
	////1
	tx.Commit()
	return &user, nil
}

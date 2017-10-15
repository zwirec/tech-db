package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/zwirec/tech-db/db"
	"github.com/qiangxue/fasthttp-routing"
	"log"
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

func (u *UserUpdate) UpdateProfile(ctx *routing.Context) (*User, error) {
	dba := database.DB
	tx := dba.MustBegin()
	user := User{}
	if err := tx.QueryRowx(`SELECT id FROM "user" u
									WHERE lower(u.nickname) = lower($1)`,
		ctx.Param("nickname")).Scan(new (int)); err != nil {
		log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "user"}

	}
	//log.Println(u.Email)
	if err := tx.QueryRowx(`UPDATE "user" SET fullname = COALESCE($1, fullname),
													email = COALESCE($2, email),
													about = COALESCE($3, about)
									WHERE nickname = $4 RETURNING nickname, fullname, email, about`,
							u.Fullname, u.Email, u.About, ctx.Param("nickname")).StructScan(&user); err != nil {
		//log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_ALREADY_EXISTS, Model: "user"}
	}
	log.Println(tx.Commit())
	return &user, nil
}

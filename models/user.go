package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"regexp"
	"github.com/zwirec/tech-db/db"
	"log"
)

var regexpUsername = regexp.MustCompile(`[a-zA-Z0-9_.]*`)

var forumUsernameSlug = []validation.Rule{
	validation.Required,
	validation.Match(regexpUsername),
}

//easyjson:json
type User struct {
	Nickname string  `json:"nickname"`
	Fullname string  `json:"fullname"`
	Email    string  `json:"email"`
	About    string  `json:"about"`
}

func (u *User) GetProfile() (*User, error) {
	dba := database.DB
	user := User{}
	tx := dba.MustBegin()
	if err := tx.QueryRowx(`SELECT nickname, fullname, email, about FROM "user" u
									WHERE lower(u.nickname) = lower($1)`, u.Nickname).StructScan(&user); err != nil {
		tx.Rollback()
		log.Println(err)
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "user"}
	}
	tx.Commit()
	return &user, nil
}

//easyjson:json
type Users []*User

func (u *User) Validate() error {
	return validation.ValidateStruct(u,
		validation.Field(&u.Nickname, forumUsernameSlug...),
		validation.Field(&u.Email, validation.Required, is.Email),
		validation.Field(&u.Fullname, validation.Required),
	)
}

func (u *User) Create() (*Users, error) {
	dba := database.DB
	user := User{}
	users := Users{}
	tx := dba.MustBegin()
	rows, _ := tx.Queryx(`SELECT fullname, nickname, about, email
							FROM "user"
							WHERE "user".nickname = $1 OR "user".email = $2;`,
		u.Nickname, u.Email)

	if exist := rows.Next(); !exist {
		rows := tx.QueryRowx(`INSERT INTO "user" (fullname, nickname, about, email) VALUES ($1, $2, $3, $4)
							RETURNING fullname, nickname, about, email`, u.Fullname, u.Nickname, u.About, u.Email)
		if err := rows.StructScan(&user); err != nil {
			log.Println(err)
		}
		tx.Commit()
		return &Users{&user}, nil
	} else {
		for exist {
			u := User{}
			rows.Scan(&u.Fullname, &u.Nickname, &u.About, &u.Email)
			rows.StructScan(&u)
			users = append(users, &u)
			exist = rows.Next()
		}
		//fmt.Printf("%+v\n", *users[0])
		rows.Close()
		tx.Commit()
		return &users, &database.DBError{Type: database.ERROR_ALREADY_EXISTS, Model: "users"}
	}
}

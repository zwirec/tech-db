package models

import (
	"log"
	"regexp"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/zwirec/tech-db/db"
)

var regexpUsername = regexp.MustCompile(`[a-zA-Z0-9_.]*`)

var forumUsernameSlug = []validation.Rule{
	validation.Required,
	validation.Match(regexpUsername),
}

//easyjson:json
type User struct {
	Nickname string `json:"nickname"`
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
	About    string `json:"about"`
}

func (u *User) GetProfile() (*User, *Error) {
	user := User{}
	tx := database.DB
	log.Println(u.Nickname)
	if err := tx.QueryRow(`SELECT nickname::text, fullname, email::text, about FROM "user" u
									WHERE u.nickname = $1`, u.Nickname).
		Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About); err != nil {
		//tx.Rollback()
		log.Println(err)
		return nil, &Error{Type: ErrorNotFound}
	}
	//tx.Commit()
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

func (u *User) Create() (*Users, *Error) {
	user := User{}
	users := Users{}
	tx, err := database.DB.Begin()

	if err != nil {
		log.Fatal(err)
	}

	log.Println(u.Nickname)
	rows, err := tx.Query(`SELECT fullname, nickname::text, about, email::text
							FROM "user"
							WHERE "user".nickname = $1 OR "user".email = $2;`,
		u.Nickname, u.Email)

	if err != nil {
		log.Fatal(err)
	}

	if exist := rows.Next(); !exist {
		row := tx.QueryRow(`INSERT INTO "user" (fullname, nickname, about, email) VALUES ($1, $2, $3, $4)
							RETURNING fullname, nickname::text, about, email::text`, u.Fullname, u.Nickname, u.About, u.Email)

		if err := row.Scan(&user.Fullname, &user.Nickname, &user.About, &user.Email); err != nil {
			log.Println(err)
		}
		tx.Commit()
		return &Users{&user}, nil
	} else {
		for exist {
			u := User{}
			rows.Scan(&u.Fullname, &u.Nickname, &u.About, &u.Email)
			//rows.Scan(&u)
			users = append(users, &u)
			exist = rows.Next()
		}
		defer rows.Close()
		tx.Commit()
		return &users, &Error{Type: ErrorAlreadyExists}
	}
}

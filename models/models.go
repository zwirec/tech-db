package models

import (
	"log"

	"github.com/zwirec/tech-db/db"
)

type Model struct {
}

type Creatable interface {
	Create() (interface{}, *Error)
}

//go:generate easyjson -snake_case ../models/

func Clear() error {
	tx, _ := database.DB.Begin()
	if _, err := tx.Exec(`TRUNCATE TABLE forum, thread, "user", post RESTART IDENTITY CASCADE`);
		err != nil {
		tx.Rollback()
		log.Println(err)
	}
	tx.Commit()
	database.DB.Reset()
	return nil
}

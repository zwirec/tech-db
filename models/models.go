package models

import (
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
		//1
	}
	tx.Commit()
	database.DB.Reset()
	return nil
}

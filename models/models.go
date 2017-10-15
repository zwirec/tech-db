package models

import (
	"log"
	"github.com/zwirec/tech-db/db"
)


//go:generate easyjson -snake_case ../models/

func Clear() error {
	tx := database.DB.MustBegin()
	if _, err := tx.Exec(`TRUNCATE TABLE forum, thread, "user" RESTART IDENTITY CASCADE`);
		err != nil {
		tx.Rollback()
		log.Println(err)
	}
	tx.Commit()
	return nil
}
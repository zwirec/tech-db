package models

import (
	"github.com/zwirec/tech-db/db"
	"log"
)

//easyjson:json
type Status struct {
	Forum  int32
	Post   int64
	Thread int32
	User   int32
}

func (s *Status) Status() (*Status, error) {
	tx := database.DB.MustBegin()
	//log.Println("hello")
	if err := tx.QueryRowx(`SELECT (SELECT count(f.*) FROM forum f) as forum,
								(SELECT count(t.*) FROM thread t) as thread,
								(SELECT count(u.*) FROM "user" u) as user,
								(SELECT count(p.*) FROM post p) as post;`).StructScan(s); err != nil {
		log.Println(err)
		tx.Rollback()
	}
	tx.Commit()
	return s, nil
}

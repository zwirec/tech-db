package models

import (
	"github.com/zwirec/tech-db/db"
)

//easyjson:json
type Status struct {
	Forum  int32
	Post   int64
	Thread int32
	User   int32
}

func (s *Status) Status() (*Status, error) {
	tx := database.DB
	////1
	if err := tx.QueryRow(`SELECT (SELECT count(f.*) FROM forum f) as forum,
								(SELECT count(t.*) FROM thread t) as thread,
								(SELECT count(u.*) FROM "user" u) as user,
								(SELECT count(p.*) FROM post p) as post;`).
		Scan(&s.Forum, &s.Thread, &s.User, &s.Post);
		err != nil {
		//1
		//tx.Rollback()
	}
	//tx.Commit()
	return s, nil
}

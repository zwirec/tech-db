package models

import (
	"github.com/zwirec/tech-db/db"
	"log"
	"github.com/qiangxue/fasthttp-routing"
)

//easyjson:json
type ThreadUpdate struct {
	Message *string
	Title   *string
}

func (t *ThreadUpdate) Update(ctx *routing.Context) (*Thread, error) {
	tx := database.DB.MustBegin()
	thread := Thread{}
	var id int

	if err := tx.QueryRowx(
		`UPDATE thread SET title = COALESCE($1, title),
							message = COALESCE($2, message)
							WHERE CASE WHEN $3::INT IS NOT NULL
							THEN id = $3
							ELSE slug = $4
							END	RETURNING id`,
							t.Title,
							t.Message,
							ctx.Get("thread_id"),
							ctx.Get("thread_slug")).
							Scan(&id);
		err != nil {
		//log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "thread"}
	}

	if err := tx.QueryRowx(`SELECT t.id, t.slug, t.title,
										t.message, f.slug as forum, u.nickname as author,
										t.created, t.votes
									FROM thread t
									JOIN forum f ON (t.forum_id = f.id)
									JOIN "user" u ON (t.owner_id = u.id)
									WHERE t.id = $1`, id).
									StructScan(&thread);
		err != nil {
			log.Println(err)
			tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_ALREADY_EXISTS, Model: "user"}
	}

	tx.Commit()
	return &thread, nil
}

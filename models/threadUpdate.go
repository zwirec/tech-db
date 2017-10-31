package models

import (
	"log"

	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/db"
)

//easyjson:json
type ThreadUpdate struct {
	Message *string
	Title   *string
}

func (t *ThreadUpdate) Update(ctx *routing.Context) (*Thread, *Error) {
	tx, _ := database.DB.Begin()
	thr := Thread{}
	var id int

	if err := tx.QueryRow(
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
		return nil, &Error{Type: ErrorNotFound}
	}

	if err := tx.QueryRow(`SELECT t.id, t.slug::text, t.title,
										t.message, t.forum_slug::text as forum, t.owner_nickname::text as author,
										t.created, t.votes
									FROM thread t
									WHERE t.id = $1`, id).
		Scan(&thr.ID, &thr.Slug, &thr.Title, &thr.Message, &thr.Forum,
		&thr.Author, &thr.Created, &thr.Votes);
		err != nil {
		log.Println(err)
		tx.Rollback()
		return nil, &Error{Type: ErrorAlreadyExists}
	}

	tx.Commit()
	return &thr, nil
}

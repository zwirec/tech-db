package models

import (
	"log"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/db"
)

//easyjson:json
type Vote struct {
	Nickname string
	Voice    int8
}

func (v *Vote) Vote(ctx *routing.Context) (*Thread, *Error) {

	tx, _ := database.DB.Begin()
	thr := Thread{}
	if err := tx.QueryRow(`INSERT INTO votes (user_id, thread_id, voice) SELECT
                                                (
                                                  SELECT id
                                                  FROM "user" u
                                                  WHERE nickname = $1) AS user_id,
                                                (
                                                  SELECT t.id
                                                  FROM thread t
                                                  WHERE
                                                    CASE WHEN $2::int IS NOT NULL
                                                      THEN t.id = $2::int
                                                    ELSE
                                                      lower(t.slug) = lower($3) END
                                                  )             AS thread_id,
                                                $4                     AS voice
												ON CONFLICT (user_id, thread_id)
  												DO UPDATE SET voice = EXCLUDED.voice
												RETURNING thread_id`,
		v.Nickname, ctx.Get("thread_id"), ctx.Get("thread_slug"), v.Voice).Scan(new(int)); err != nil {
		log.Println(err)
		tx.Rollback()
		return nil, &Error{Type: ErrorNotFound}
	}

	if err := tx.QueryRow(`SELECT t.id, t.slug::text, t.title, t.message, t.forum_slug::text as forum, t.owner_nickname::text as author, t.created, t.votes
						FROM thread t
						WHERE
						CASE WHEN $1::int IS NOT NULL THEN t.id = $1
						ELSE t.slug = $2
						END`, ctx.Get("thread_id"), ctx.Get("thread_slug")).
		Scan(&thr.ID, &thr.Slug, &thr.Title, &thr.Message, &thr.Forum, &thr.Author, &thr.Created, &thr.Votes);
		err != nil {
		log.Println(err)
		tx.Rollback()
		return nil, &Error{Type: ErrorAlreadyExists}
	}
	tx.Commit()
	return &thr, nil
}

func (v *Vote) Validate() error {
	return validation.ValidateStruct(v,
		validation.Field(v.Nickname, validation.Required),
		validation.Field(v.Voice, validation.Required, validation.In(1, -1)),
	)
}

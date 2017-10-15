package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/db"
	"log"
)

//easyjson:json
type Vote struct {
	Nickname string
	Voice    int8
}

func (v *Vote) Vote(ctx *routing.Context) (*Thread, error) {

	tx := database.DB.MustBegin()
	thread := Thread{}
	if err := tx.QueryRowx(`INSERT INTO votes (user_id, thread_id, voice) SELECT
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
		v.Nickname, ctx.Get("thread_id"), ctx.Get("thread_slug"), v.Voice).Scan(new (int)); err != nil {
		//log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "thread"}
	}

	if err := tx.QueryRowx(`SELECT t.id, t.slug, t.title, t.message, f.slug as forum, u.nickname as author, t.created, t.votes
						FROM thread t
						JOIN forum f ON (t.forum_id = f.id)
						JOIN "user" u ON (t.owner_id = u.id)
						WHERE
						CASE WHEN $1::int IS NOT NULL THEN t.id = $1
						ELSE t.slug = $2
						END`, ctx.Get("thread_id"), ctx.Get("thread_slug")).StructScan(&thread); err != nil {
		log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_ALREADY_EXISTS, Model: "voice"}
	}
	tx.Commit()
	return &thread, nil
}

func (v *Vote) Validate() error {
	return validation.ValidateStruct(v,
		validation.Field(v.Nickname, validation.Required),
		validation.Field(v.Voice, validation.Required, validation.In(1, -1)),
	)
}

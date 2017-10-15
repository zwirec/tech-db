package models

import (
	"time"
	"github.com/go-ozzo/ozzo-validation"
	"regexp"
	"github.com/zwirec/tech-db/db"
	"log"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/jmoiron/sqlx"
)

var threadRuleSlug = []validation.Rule{
	validation.Required,
	validation.Match(regexp.MustCompile(`^(\d|\w|-|_)*(\w|-|_)(\d|\w|-|_)*$`)),
}

//easyjson:json
type Thread struct {
	ID      *int32
	Author  string
	Forum   string
	Message string
	Slug    *string `json:",omitempty"`
	Title   string
	Votes   int32
	Created time.Time
}

func (t *Thread) Posts(ctx *routing.Context) (Posts, error) {
	tx := database.DB.MustBegin()

	if err := tx.QueryRowx(`SELECT id
						FROM thread t
						WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1
							  ELSE lower(t.slug) = lower($2)
							  END`, ctx.Get("thread_id"), ctx.Get("thread_slug")).
		Scan(new(int)); err != nil {
		//log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "thread"}
	}

	var rows *sqlx.Rows
	var err error
	posts := Posts{}
	limit := ctx.Get("limit").(string)
	sort := ctx.Get("sort").(string)
	if ctx.Get("sort_type") == "flat" || ctx.Get("sort_type") == "" {
		if rows, err = tx.Queryx(
			`SELECT p.id, p.message, p.thread_id as thread, f.slug as forum, u.nickname as author,
					p.created, p.isedited, p.parent FROM post p
					JOIN thread t ON (p.thread_id = t.id)
					JOIN forum f ON (t.forum_id = f.id)
					JOIN "user" u ON (u.id = p.owner_id)
					WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1
					ELSE lower(t.slug) = lower($2)
					END
					AND
					CASE WHEN $3 > -1 THEN
						CASE WHEN $4 = 'DESC'
							THEN p.id < $3::int
						ELSE p.id > $3::int
						END
					ELSE p.id > $3
					END
					ORDER BY p.id `+
				sort+
				` LIMIT `+
				limit+ `;`,
			ctx.Get("thread_id"),
			ctx.Get("thread_slug"),
			ctx.Get("since"),
			ctx.Get("sort"));
			err != nil {
			log.Println(err)
			tx.Rollback()
			return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "thread"}
		}

	} else if ctx.Get("sort_type") == "tree" {
		//log.Println("tree")
		if rows, err = tx.Queryx(
			`SELECT p.id, p.message, p.thread_id as thread, f.slug as forum, u.nickname as author,
					p.created, p.isedited, p.parent
					FROM post p
					JOIN thread t ON (p.thread_id = t.id)
					JOIN forum f ON (t.forum_id = f.id)
					JOIN "user" u ON (u.id = p.owner_id)
					WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1
					ELSE lower(t.slug) = lower($2)
					END
					AND
					CASE WHEN $3 > -1 AND $4 = 'DESC'
						THEN path < (SELECT p.path FROM post p WHERE p.id = $3)
						WHEN $3 > -1 AND $4 = 'ASC'
						THEN path > (SELECT p.path FROM post p WHERE p.id = $3)
					ELSE parent > $3
					END
					ORDER BY string_to_array(subltree(p.path, 0, 1)::text,'.')::integer[] `+
				sort+
				`, string_to_array(p.path::text,'.')::integer[] `+ sort+ ` LIMIT `+
				limit+ `;`,
			ctx.Get("thread_id"),
			ctx.Get("thread_slug"),
			ctx.Get("since"),
			ctx.Get("sort"));
			err != nil {
			log.Println(err)
			tx.Rollback()
			return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "thread"}
		}
	} else {
		if rows, err = tx.Queryx(
			`SELECT p.id, p.message, p.thread_id as thread, f.slug as forum, u.nickname as author,
					p.created, p.isedited, p.parent
					FROM post p
					JOIN thread t ON (p.thread_id = t.id)
					JOIN forum f ON (t.forum_id = f.id)
					JOIN "user" u ON (u.id = p.owner_id)
					WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1
					ELSE lower(t.slug) = lower($2)
					END
					AND subltree(p.path, 0, 1) IN (
						SELECT p1.path
						FROM post p1
						WHERE nlevel(p1.path) = 1 AND p1.thread_id = t.id and
						CASE WHEN $3 > -1 AND $4 = 'DESC'
          						THEN path < (SELECT p2.path FROM post p2 WHERE p2.id = $3)
        					WHEN $3 > -1 AND $4 = 'ASC'
          						THEN path > (SELECT p2.path FROM post p2 WHERE p2.id = $3)
						ELSE p1.id > $3
						END
						ORDER BY string_to_array(p1.path::text,'.')::integer[] `+ sort+
				` LIMIT `+ limit+
				`)
			ORDER BY string_to_array(subltree(p.path, 0, 1)::text,'.')::integer[] `+
				sort+
				`, string_to_array(p.path::text,'.')::integer[] `+
				sort+ `;`,
			ctx.Get("thread_id"),
			ctx.Get("thread_slug"),
			ctx.Get("since"),
			ctx.Get("sort"));
			err != nil {
			log.Println(err)
			tx.Rollback()
			return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "thread"}
		}
	}

	defer rows.Close()

	for rows.Next() {
		post := Post{}
		rows.StructScan(&post)
		posts = append(posts, &post)
	}

	tx.Commit()
	return posts, nil
}

func (t *Thread) Details(ctx *routing.Context) (*Thread, error) {
	dba := database.DB
	tx := dba.MustBegin()
	thread := Thread{}
	if err := tx.QueryRowx(`SELECT t.id, t.slug, t.title, t.message, f.slug as forum, u.nickname as author, t.created, t.votes
						FROM thread t
						JOIN forum f ON (t.forum_id = f.id)
						JOIN "user" u ON (t.owner_id = u.id)
						WHERE CASE WHEN $1::int IS NOT NULL THEN t.id::int = $1
						ELSE lower(t.slug) = lower($2)
						END`,
		ctx.Get("thread_id"), ctx.Get("thread_slug")).StructScan(&thread); err != nil {
		//log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "thread"}
	}
	tx.Commit()
	return &thread, nil
}

func IsDefined() bool {
	return false
}

//easyjson:json
type Threads []*Thread

func (thr *Thread) Validate() error {
	return validation.ValidateStruct(thr,
		validation.Field(&thr.Message, validation.Required),
		validation.Field(&thr.Author, validation.Required),
		validation.Field(&thr.Title, validation.Required),
		//validation.Field(&thr.Slug, threadRuleSlug...),
	)
}

func (thr *Thread) Create() (*Thread, error) {
	dba := database.DB
	tx := dba.MustBegin()
	var user_id int64
	if err := tx.QueryRowx(`SELECT u.id
						FROM "user" u
						WHERE u.nickname = $1;`,
		thr.Author).
		Scan(&user_id); err != nil {
		tx.Rollback()
		log.Println(err)
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "user"}
	}

	var forum_id int64
	var forum_slug string

	if err := tx.QueryRowx(`SELECT f.id, f.slug
						FROM forum f
						WHERE f.slug = $1;`,
		thr.Forum).
		Scan(&forum_id, &forum_slug); err != nil {
		tx.Rollback()
		log.Println(err)
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "forum"}
	}

	thr.Forum = forum_slug

	if err := tx.QueryRowx(`SELECT t.id, t.title, t.slug, t.message, t.created, t.votes, f.slug as forum, u.nickname as author FROM thread t
									JOIN forum f ON (t.forum_id = f.id)
									JOIN "user" u ON (t.owner_id = u.id)
									WHERE lower(t.slug) = lower($1);`, thr.Slug).
		StructScan(thr);
		err == nil {
		tx.Rollback()
		//log.Println(err)
		return thr, &database.DBError{Type: database.ERROR_ALREADY_EXISTS, Model: "thread"}
	} else {
		//log.Println(err)
	}

	res := tx.QueryRowx(`INSERT INTO thread (slug, title, message, created, owner_id, forum_id)
									VALUES (nullif($1, ''), $2, $3, $4, $5, $6)
								    RETURNING id, slug, title, message, created, votes`,
		thr.Slug, thr.Title, thr.Message, thr.Created, user_id, forum_id)

	if err := res.StructScan(thr); err != nil {
		log.Println(err)
		tx.Rollback()
		return thr, &database.DBError{Model: "thread", Type: database.ERROR_ALREADY_EXISTS}
	}

	tx.Commit()
	return thr, nil

}

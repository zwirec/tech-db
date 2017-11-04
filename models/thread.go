package models

import (
	"regexp"
	"time"

	"sync"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/jackc/pgx"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/db"
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

var rowsPool = sync.Pool{New: func() interface{} {
	return &pgx.Rows{}
}}

func (t *Thread) Posts(ctx *routing.Context) (Posts, *Error) {
	tx := database.DB
	var threadId int
	if err := tx.QueryRow(`SELECT id FROM thread t WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1 ELSE t.slug = $2 END`,
		ctx.Get("thread_id"),
		ctx.Get("thread_slug")).Scan(&threadId); err != nil {
		//1
		//tx.Rollback()
		return nil, &Error{Type: ErrorNotFound}
	}

	//if err := tx.QueryRowx(`SELECT id
	//					FROM thread t
	//					WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1
	//						  ELSE t.slug = $2
	//						  END`, ctx.Get("thread_id"), ctx.Get("thread_slug")).
	//	Scan(new(int)); err != nil {
	//	////1
	//	tx.Rollback()
	//	return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "thread"}
	//}

	var rows = rowsPool.Get().(*pgx.Rows)

	defer rowsPool.Put(rows)

	var err error
	//posts := PostsPool.Get().(Posts)
	posts := Posts{}
	//defer PostsPool.Put(posts)
	limit := ctx.Get("limit").(string)
	sort := ctx.Get("sort").(string)

	if ctx.Get("sort_type") == "flat" || ctx.Get("sort_type") == "" {
		if rows, err = tx.Query(
			`SELECT p.id, p.message, p.thread_id as thread, p.forum_slug::text as forum, p.owner_nickname::text as author,
					p.created, p.isedited, p.parent FROM post p
					WHERE p.thread_id = $1 AND
					CASE WHEN $2 > -1 THEN
						CASE WHEN $3 = 'DESC'
							THEN p.id < $2::int
						ELSE p.id > $2::int
						END
					ELSE TRUE
					END
					ORDER BY p.id `+
				sort+
				`,p.thread_id `+ sort+ ` LIMIT `+
				limit+ `;`,
			threadId,
			ctx.Get("since"),
			ctx.Get("sort"));
			err != nil {
			//1
			//tx.Rollback()
			return nil, &Error{Type: ErrorNotFound}
		}

	} else if ctx.Get("sort_type") == "tree" {
		if rows, err = tx.Query(
			`SELECT
                      p.id,
                      p.message,
                      p.thread_id      AS thread,
                      p.forum_slug::text     AS forum,
                      p.owner_nickname::text AS author,
                      p.created,
                      p.isedited,
                      p.parent
                    FROM post p
                    WHERE p.thread_id = $1
                      AND
                      CASE WHEN $2::int > -1
                        THEN
                          CASE WHEN $3 = 'DESC'
                            THEN
                              path < (
                                SELECT p1.path
                                FROM post p1
                                WHERE p1.id = $2)
                          WHEN $3 = 'ASC'
                            THEN path > (
                              SELECT p1.path
                              FROM post p1
                              WHERE p1.id = $2)
							ELSE TRUE
                          END
						ELSE
                        TRUE
                      END
                    ORDER BY path `+ sort+ `, p.thread_id `+ sort+ ` LIMIT `+ limit+ `;`,
			threadId,
			ctx.Get("since"),
			ctx.Get("sort"));
			err != nil {
			//1
			//tx.Rollback()
			return nil, &Error{Type: ErrorNotFound}
		}
	} else {
		if rows, err = tx.Query(
			`SELECT
                  p.id,
                  p.message,
                  p.thread_id      AS thread,
                  p.forum_slug::text     AS forum,
                  p.owner_nickname::text AS author,
                  p.created,
                  p.isedited,
                  p.parent
                FROM post p
                  JOIN (
                         SELECT
                           p1.path,
                           p1.thread_id
                         FROM post p1
                         WHERE p1.parent = 0 AND p1.thread_id = $1::int AND
                               CASE WHEN $2 > -1
                                 THEN
                                   CASE WHEN $3 = 'DESC'
                                     THEN
                                       p1.path < (
                                         SELECT p2.path
                                         FROM post p2
                                         WHERE p2.id = $2::int)

                                   ELSE p1.path > (
                                     SELECT p2.path
                                     FROM post p2
                                     WHERE p2.id = $2::int)
                                   END
                               ELSE p1.id > 0
                               END
                         ORDER BY p1.path `+ sort+ `
                         ,p1.thread_id `+ sort+ ` LIMIT `+ limit+ `) p1 ON p.path && p1.path
				WHERE p.thread_id = $1::int
                ORDER BY p.path `+ sort+ `, p.thread_id `+ sort+ `;`,
			threadId,
			ctx.Get("since"),
			ctx.Get("sort"));
			err != nil {
			//1
			//tx.Rollback()
			return nil, &Error{Type: ErrorNotFound}
		}
	}

	for rows.Next() {
		post := Post{}
		rows.Scan(&post.ID, &post.Message, &post.Thread, &post.Forum, &post.Author, &post.Created, &post.IsEdited, &post.Parent)
		posts = append(posts, &post)
	}

	//tx.Commit()
	return posts, nil
}

func (t *Thread) Details(ctx *routing.Context) (*Thread, *Error) {
	tx := database.DB
	thr := Thread{}
	if err := tx.QueryRow(`SELECT t.id, t.slug::text, t.title, t.message, t.forum_slug::text as forum,
										t.owner_nickname::text as author, t.created, t.votes
						FROM thread t
						WHERE CASE WHEN $1::int IS NOT NULL THEN t.id::int = $1
						ELSE t.slug = $2
						END`,
		ctx.Get("thread_id"), ctx.Get("thread_slug")).
		Scan(&thr.ID, &thr.Slug, &thr.Title, &thr.Message, &thr.Forum, &thr.Author, &thr.Created, &thr.Votes);
		err != nil {
		//1
		//tx.Rollback()
		return nil, &Error{Type: ErrorNotFound}
	}
	//tx.Commit()
	return &thr, nil
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

func (thr *Thread) Create() (*Thread, *Error) {
	tx, _ := database.DB.Begin()

	var user_id int64

	if err := tx.QueryRow(`SELECT u.id
						FROM "user" u
						WHERE u.nickname = $1;`,
		thr.Author).
		Scan(&user_id);
		err != nil {
		tx.Rollback()
		//1
		return nil, &Error{Type: ErrorNotFound}
	}

	var forumId int64
	var forumSlug string

	if err := tx.QueryRow(`SELECT f.id, f.slug::text
									FROM forum f
									WHERE f.slug = $1;`,
		&thr.Forum).
		Scan(&forumId, &forumSlug);
		err != nil {
		tx.Rollback()
		//1
		return nil, &Error{Type: ErrorNotFound}
	}

	thr.Forum = forumSlug

	if err := tx.QueryRow(`SELECT t.id, t.title, t.slug::text, t.message, t.created,
										t.votes, t.forum_slug::text as forum, t.owner_nickname::text as author
									FROM thread t
									WHERE t.slug = $1;`, thr.Slug).
		Scan(&thr.ID, &thr.Title, &thr.Slug, &thr.Message, &thr.Created, &thr.Votes,
		&thr.Forum, &thr.Author);
		err == nil {
		tx.Rollback()
		//1
		return thr, &Error{Type: ErrorAlreadyExists}
	} else {
		////1
	}

	res := tx.QueryRow(`INSERT INTO thread (slug, title, message, created, owner_id, owner_nickname, forum_id, forum_slug)
									VALUES (nullif($1, ''), $2, $3, $4, $5, $6, $7, $8)
								    RETURNING id, slug::text, title, message, created, votes`,
		thr.Slug, thr.Title, thr.Message, thr.Created, user_id, thr.Author, forumId, thr.Forum)

	if err := res.Scan(&thr.ID, &thr.Slug, &thr.Title, &thr.Message, &thr.Created, &thr.Votes); err != nil {
		//1
		tx.Rollback()
		return thr, &Error{Type: ErrorAlreadyExists}
	}

	tx.Commit()
	return thr, nil

}

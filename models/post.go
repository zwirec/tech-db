package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"github.com/zwirec/tech-db/db"
	"log"
	"github.com/qiangxue/fasthttp-routing"
	"time"
)

//easyjson:json
type Post struct {
	ID       int32
	Author   string
	Created  time.Time
	Forum    string
	IsEdited bool    `json:"isEdited"`
	Message  *string
	Parent   int32
	Thread   int32
}

func (p *Post) Update() (*Post, error) {
	tx := database.DB.MustBegin()

	if err := tx.QueryRowx(`WITH new_row AS (UPDATE post p
									SET message = COALESCE($1, message),
									isedited = CASE WHEN $1 IS NOT NULL AND
												$1 != (select p1.message from post p1 WHERE p1.id = $2)
													THEN TRUE
													ELSE FALSE
												END
									WHERE p.id = $2
									RETURNING id, message, thread_id, parent, created, isedited, owner_id)
									SELECT
									  nr.id,
									  nr.message,
									  nr.thread_id AS thread,
									  nr.parent,
									  u.nickname  AS author,
									  nr.created,
									  f.slug      AS forum,
									  nr.isedited
									FROM new_row nr
									JOIN thread t ON nr.thread_id = t.id
									JOIN forum f ON t.forum_id = f.id
									JOIN "user" u ON u.id = nr.owner_id`,
		p.Message, p.ID).
		StructScan(p); err != nil {
		//log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "post"}
	}
	tx.Commit()
	return p, nil
}

func (p *Post) Details(ctx *routing.Context) (*PostFull, error) {
	tx := database.DB.MustBegin()
	postFull := PostFull{Post: &Post{}}
	related := ctx.Get("related").(map[string]bool)
	if err := tx.QueryRowx(`SELECT p.id, p.message, p.thread_id as thread, f.slug as forum,
								parent, u.nickname as author, p.created, isedited
								FROM post p
								JOIN thread t ON p.thread_id = t.id
								JOIN forum f ON t.forum_id = f.id
								JOIN "user" u ON p.owner_id = u.id
								WHERE p.id = $1`, p.ID).StructScan(postFull.Post); err != nil {
		tx.Rollback()
		//log.Println(err)
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "post"}
	}

	if related["user"] {
		postFull.Author = &User{}
		if err := tx.QueryRowx(`SELECT about, email, fullname, nickname
						FROM post p
						JOIN "user" u ON (p.owner_id = u.id)
						WHERE p.id = $1`, p.ID).
			StructScan(postFull.Author);
			err != nil {
			tx.Rollback()
			//log.Println(err)
			return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "post"}
		}
	}

	if related["thread"] {
		postFull.Thread = &Thread{}
		if err := tx.QueryRowx(`SELECT t.id, t.slug, t.title, t.message, $2::text as forum, u.nickname as author,
										t.created, t.votes
										FROM post p
										JOIN thread t ON (p.thread_id = t.id)
										JOIN "user" u ON (t.owner_id = u.id)
										WHERE p.id = $1`, p.ID, postFull.Post.Forum).
			StructScan(postFull.Thread);
			err != nil {
			tx.Rollback()
			//log.Println(err)
			return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "post"}
		}
	}

	if related["forum"] {
		postFull.Forum = &Forum{}
		if err := tx.QueryRowx(`SELECT f.slug, f.title, f.posts, f.threads, u.nickname as user
										FROM post p
										JOIN thread t ON (p.thread_id = t.id)
										JOIN forum f ON (t.forum_id = f.id)
										JOIN "user" u ON (f.owner_id = u.id)
										WHERE p.id = $1`, p.ID).
			StructScan(postFull.Forum);
			err != nil {
			tx.Rollback()
			//log.Println(err)
			return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "post"}
		}
	}
	tx.Commit()
	return &postFull, nil
}

//easyjson:json
type Posts []*Post

func (p *Post) Validate() error {
	return validation.ValidateStruct(p,
		validation.Field(&p.Author, validation.Required),
		validation.Field(&p.Message, validation.Required),
	)
}

func (ps *Posts) Create(ctx *routing.Context) (*Posts, error) {
	posts := make(Posts, 0, len(*ps))
	//fastsql.DB.DB = database.DB.DB

	var user_nickname string
	var user_id int

	var thread_id int
	var forum_slug string

	if err := database.DB.QueryRowx(`SELECT t.id, f.slug
						FROM thread t
						JOIN forum f ON (t.forum_id = f.id)
						WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1
						ELSE lower(t.slug) = lower($2)
						END`, ctx.Get("thread_id"), ctx.Get("thread_slug")).Scan(&thread_id, &forum_slug);
		err != nil {
		//log.Println(err)
		return nil, &database.DBError{Type: database.ERROR_ALREADY_EXISTS, Model: "thread"}
	}

	tx := database.DB.MustBegin()
	stmt1, err := tx.Preparex(`SELECT id, nickname FROM "user" u WHERE u.nickname = $1`)

	if err != nil {
		log.Fatal(err)
		tx.Rollback()
		return nil, err
	}

	stmt2, err := tx.Preparex(`SELECT p.thread_id FROM post p
									JOIN thread t ON (p.thread_id = t.id)
									WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1
									ELSE t.slug = $2
									END
									AND p.id = $3`)
	if err != nil {
		log.Fatal(err)
		tx.Rollback()
	}

	stmt3, err := tx.Preparex(`INSERT INTO post (message, thread_id, parent, owner_id, created)
										VALUES ($1, $2, $3, $4, $5) RETURNING id, message, thread_id as thread,
										parent, owner_id as author, created, isedited`)

	if err != nil {
		log.Println(err)
		tx.Rollback()
	}

	for _, p := range *ps {
		if err := stmt1.QueryRowx(p.Author).Scan(&user_id, &user_nickname); err != nil {
			tx.Rollback()
			log.Println(err)
			return nil, &database.DBError{Type: database.ERROR_ALREADY_EXISTS, Model: "user"}
		}

		if p.Parent != 0 {
			if err := stmt2.QueryRowx(ctx.Get("thread_id"), ctx.Get("thread_slug"), p.Parent).Scan(&thread_id);
				err != nil {
				tx.Rollback()
				//log.Println(err)
				return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "parent"}
			}
		}

		post := Post{}
		if err := stmt3.QueryRowx(p.Message, thread_id, p.Parent, user_id, ctx.Get("created")).
			StructScan(&post); err != nil {
			log.Fatal(err)
			tx.Rollback()
			return nil, err
		}
		post.Forum = forum_slug
		post.Author = user_nickname

		post.Created = post.Created
		posts = append(posts, &post)
	}
	forum := Forum{Slug: forum_slug}
	forum.UpdateCountPosts(len(*ps))

	tx.Commit()
	return &posts, nil
}
func (f *Forum) UpdateCountPosts(length int) {
	tx := database.DB.MustBegin()
	if _, err := tx.Exec("UPDATE forum SET posts = posts + $1 WHERE slug = $2", length, f.Slug); err != nil {
		log.Fatal(err)
		tx.Rollback()
	}
	tx.Commit()
}

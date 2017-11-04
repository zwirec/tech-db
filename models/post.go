package models

import (
	"log"
	"time"

	"context"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/db"
	"sync"
)

//easyjson:json
type Post struct {
	ID       int64
	Author   string
	Created  time.Time
	Forum    string
	IsEdited bool `json:"isEdited"`
	Message  *string
	Parent   int32
	Thread   int32
}

//easyjson:json
type Posts []*Post

func (p *Post) Update() (*Post, *Error) {
	tx, _ := database.DB.Begin()

	if err := tx.QueryRow(`WITH new_row AS (UPDATE post p
									SET message = COALESCE($1, message),
									isedited = CASE WHEN $1 IS NOT NULL AND
												$1 != (select p1.message from post p1 WHERE p1.id = $2)
													THEN TRUE
													ELSE FALSE
												END
									WHERE p.id = $2
									RETURNING id, message, thread_id, parent, created, isedited, owner_id, owner_nickname, forum_slug)
									SELECT
									  nr.id,
									  nr.message,
									  nr.thread_id 		AS thread,
									  nr.parent,
									  nr.owner_nickname  AS author,
									  nr.created,
									  nr.forum_slug      AS forum,
									  nr.isedited
									FROM new_row nr`,
		p.Message, p.ID).
		Scan(&p.ID, &p.Message, &p.Thread, &p.Parent, &p.Author,
		&p.Created, &p.Forum, &p.IsEdited);
		err != nil {
		//1
		tx.Rollback()
		return nil, &Error{Type: ErrorNotFound}
	}

	tx.Commit()
	return p, nil
}

func (p *Post) Details(ctx *routing.Context) (*PostFull, *Error) {
	tx := database.DB
	postFull := PostFull{Post: &Post{}}
	related := ctx.Get("related").(map[string]bool)

	if err := tx.QueryRow(`SELECT p.id, p.message, p.thread_id as thread, forum_slug::text as forum,
								p.parent, p.owner_nickname::text as author, p.created, p.isedited
								FROM post p
								WHERE p.id = $1`, p.ID).
		Scan(&postFull.Post.ID, &postFull.Post.Message, &postFull.Post.Thread,
		&postFull.Post.Forum, &postFull.Post.Parent, &postFull.Post.Author,
		&postFull.Post.Created, &postFull.Post.IsEdited);
		err != nil {
		//tx.Rollback()
		//1
		return nil, &Error{Type: ErrorNotFound}
	}

	if related["user"] {
		postFull.Author = &User{}
		if err := tx.QueryRow(`SELECT about, email::text, fullname, nickname::text
						FROM post p
						JOIN "user" u ON (p.owner_id = u.id)
						WHERE p.id = $1`, p.ID).
			Scan(&postFull.Author.About, &postFull.Author.Email, &postFull.Author.Fullname, &postFull.Author.Nickname);
			err != nil {
			//tx.Rollback()
			//1
			return nil, &Error{Type: ErrorNotFound}
		}
	}

	if related["thread"] {
		postFull.Thread = &Thread{}
		if err := tx.QueryRow(`SELECT t.id, t.slug::text, t.title, t.message, $2::text as forum, t.owner_nickname::text as author,
										t.created, t.votes
										FROM post p
										JOIN thread t ON (p.thread_id = t.id)
										WHERE p.id = $1`, p.ID, postFull.Post.Forum).
			Scan(&postFull.Thread.ID, &postFull.Thread.Slug, &postFull.Thread.Title,
			&postFull.Thread.Message, &postFull.Thread.Forum, &postFull.Thread.Author,
			&postFull.Thread.Created, &postFull.Thread.Votes);
			err != nil {
			//tx.Rollback()
			//1
			return nil, &Error{Type: ErrorNotFound}
		}
	}

	if related["forum"] {
		postFull.Forum = &Forum{}
		if err := tx.QueryRow(`SELECT f.slug::text, f.title, f.posts, f.threads, f.owner_nickname::text as user
										FROM post p
										JOIN forum f ON (p.forum_slug = f.slug)
										WHERE p.id = $1`, p.ID).
			Scan(&postFull.Forum.Slug, &postFull.Forum.Title, &postFull.Forum.Posts,
			&postFull.Forum.Threads, &postFull.Forum.User);
			err != nil {
			//tx.Rollback()
			//1
			return nil, &Error{Type: ErrorNotFound}
		}
	}

	//postFull.Post.Created = postFull.Post.Created.In(time.Local)

	//tx.Commit()
	return &postFull, nil
}

func (p *Post) Validate() error {
	return validation.ValidateStruct(p,
		validation.Field(&p.Author, validation.Required),
		validation.Field(&p.Message, validation.Required),
	)
}


var PostsPool = sync.Pool{New: func() interface{} {
	return Posts{}
}}

var userPool = sync.Pool{New: func() interface{} {
	return &User{}
}}

func (ps *Posts) Create(ctx *routing.Context) (Posts, *Error) {
	posts := make(Posts, 0, len(*ps))

	user := User{}
	//defer userPool.Put(user)

	var userNickname string
	var userId int

	var threadId int32
	var forumSlug string

	tx, _ := database.DB.Begin()

	_, err := tx.Prepare("stmt1", `SELECT * FROM "user" u WHERE u.nickname = $1`)

	//if len(*ps) != 0 {
	//	if err := tx.QueryRow(`SELECT * FROM "user" u WHERE u.nickname = $1`, (*ps)[0].Author).
	//		Scan(&userId, &userNickname, &user.Fullname, &user.Email, &user.About);
	//		err != nil {
	//		tx.Rollback()
	//		//1
	//		return nil, &Error{Type: ErrorNotFound}
	//	}
	//}

	if err != nil {
		//1
	}

	_, err = database.DB.Prepare("users_forum", `INSERT INTO users_forum (forum_slug, nickname, fullname, email, about)
												VALUES($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`)
	if err != nil {
		//1
	}

	if err := tx.QueryRow(`SELECT t.id, t.forum_slug::text
						FROM thread t
						WHERE CASE WHEN $1::int IS NOT NULL THEN t.id = $1
						ELSE t.slug = $2
						END`, ctx.Get("thread_id"), ctx.Get("thread_slug")).Scan(&threadId, &forumSlug);
		err != nil {
		//1
		tx.Rollback()
		return nil, &Error{Type: ErrorNotFound}
	}

	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Prepare("stmt2", `SELECT p.thread_id FROM post p
									JOIN thread t ON (p.thread_id = t.id)
									AND CASE WHEN $1::int IS NOT NULL THEN t.id = $1
									ELSE t.slug = $2
									END
									WHERE p.id = $3`)
	if err != nil {
		log.Fatal(err)
		tx.Rollback()
		//tx2.Rollback()
	}

	ids := make([]int64, len(*ps)+1)

	_, err = database.DB.Prepare("stmt3",
		`INSERT INTO post (id, message, thread_id, parent,
								owner_id, owner_nickname, forum_slug, created)
										VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`)

	batchPosts := database.DB.BeginBatch()
	batchUsrForum := database.DB.BeginBatch()

	if err != nil {
		//1
		tx.Rollback()
		//tx2.Rollback()
	}

	res := tx.QueryRow(`SELECT array_agg(nextval(pg_get_serial_sequence('post', 'id'))) FROM generate_series(1, $1);`,
		len(*ps))

	if err != nil {
		tx.Rollback()
		//tx2.Rollback()
		//tx2.Commit()
		log.Fatal(err)
	}

	if err := res.Scan(&ids); err != nil {
		//1
	}

	//log.Printf("%+v", ids)

	for i, p := range *ps {
		if err := tx.QueryRow(`stmt1`, p.Author).
			Scan(&userId, &userNickname, &user.Fullname, &user.Email, &user.About);
			err != nil {
			tx.Rollback()
			//1
			return nil, &Error{Type: ErrorNotFound}
		}
		_, err := database.DB.Exec("users_forum", forumSlug, userNickname, user.Fullname, user.Email, user.About)

		if err != nil {
			//1
			tx.Rollback()
		}


		//batchUsrForum.Queue("users_forum", []interface{}{forumSlug, userNickname, user.Fullname, user.Email, user.About},
		//					nil, nil)
		////1

		if p.Parent != 0 {
			if err := tx.QueryRow("stmt2", ctx.Get("thread_id"), ctx.Get("thread_slug"), p.Parent).Scan(&threadId);
				err != nil {
				tx.Rollback()
				//1
				return nil, &Error{Type: ErrorAlreadyExists}
			}
		}

		post := Post{}
		batchPosts.Queue("stmt3", []interface{}{ids[i], p.Message, threadId, p.Parent,
			userId, userNickname, forumSlug, ctx.Get("created")}, nil, nil)

		post.Forum = forumSlug
		post.Author = userNickname
		post.ID = ids[i]
		post.Thread = threadId
		post.Message = p.Message
		post.Parent = p.Parent
		t, _ := time.Parse("2006-01-02T15:04:05.000000Z", ctx.Get("created").(string))
		post.Created = t

		posts = append(posts, &post)
	}

	if err := batchPosts.Send(context.Background(), nil); err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	if err := batchUsrForum.Send(context.Background(), nil); err != nil {
		tx.Rollback()
		log.Fatal()
	}

	_, err = batchPosts.ExecResults()
	_, err = batchUsrForum.ExecResults()

	if err != nil {
		//1
	}

	if err := batchPosts.Close(); err != nil {
		//1
	}

	if err := batchUsrForum.Close(); err != nil {
		//1
	}

	//database.DB.Deallocate("stmt1")
	//database.DB.Deallocate("stmt2")
	//database.DB.Deallocate("stmt3")
	forum := Forum{Slug: forumSlug}

	forum.UpdateCountPosts(len(*ps))
	//database.DB.Reset()
	tx.Commit()

	//if database.DB.Stat().CurrentConnections > 10 {
	//	database.DB.Reset()
	//}

	return posts, nil
}

func (f *Forum) UpdateCountPosts(length int) {
	tx, err := database.DB.Begin()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := tx.Exec("UPDATE forum SET posts = posts + $1 WHERE slug = $2", length, f.Slug); err != nil {
		log.Fatal(err)
		tx.Rollback()
	}
	tx.Commit()
}

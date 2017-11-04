package models

import (
	"log"
	"regexp"

	"sync"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/db"
)

var regexpForumSlug = regexp.MustCompile(`^(\d|\w|-|_)*(\w|-|_)(\d|\w|-|_)*$`)

var forumRuleSlug = []validation.Rule{
	validation.Required,
	validation.Match(regexpForumSlug),
}

//easyjson:json
type Forum struct {
	Posts   int32  `json:"posts"`
	Slug    string `json:"slug"`
	Threads int32  `json:"threads"`
	Title   string `json:"title"`
	User    string `json:"user"`
}

var UsersPool = sync.Pool{New: func() interface{} {
	return Users{}
}}



func (f *Forum) Users(ctx *routing.Context) (Users, *Error) {
	users := UsersPool.Get().(Users)
	tx := database.DB

	limit := ctx.Get("limit").(string)
	sort := ctx.Get("sort").(string)

	defer UsersPool.Put(users)
	if err := tx.QueryRow(`SELECT id FROM forum f WHERE f.slug = $1`, ctx.Get("forum_slug")).
		Scan(new(int)); err != nil {
		return nil, &Error{Type: ErrorNotFound}
	}

	rows, err := tx.Query(`SELECT nickname, fullname, email, about
								FROM users_forum
								WHERE forum_slug = $1 AND
								CASE WHEN $3 = 'ASC' THEN
						  		nickname > $2
						  		ELSE CASE WHEN $2 != '' THEN
								nickname < $2
								ELSE TRUE
								END
						  		END
								ORDER BY nickname ` + sort +
		` LIMIT ` + limit, ctx.Get("forum_slug"), ctx.Get("since"), sort)

	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {
		user := User{}
		rows.Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About)
		users = append(users, &user)
	}

	return users, nil
}

func (f *Forum) Select() (*Forum, *Error) {
	tx := database.DB
	err := tx.QueryRow(`SELECT slug::text, title, posts, threads,
								f.owner_nickname::text as user FROM forum f
								WHERE f.slug = $1`,
		f.Slug).Scan(&f.Slug, &f.Title, &f.Posts, &f.Threads, &f.User)
	if err != nil {
		//1
		//tx.Rollback()
		return nil, &Error{Type: ErrorNotFound}
	}
	//tx.Commit()
	return f, nil
}

func (f *Forum) Validate() error {
	return validation.ValidateStruct(f,
		validation.Field(&f.Slug, forumRuleSlug...),
		validation.Field(&f.Title, validation.Required),
		validation.Field(&f.User, validation.Required),
	)
}

func (f *Forum) Create() (*Forum, *Error) {
	tx, _ := database.DB.Begin()

	var id int

	if err := tx.QueryRow(`SELECT id, nickname::text
									FROM "user"
									WHERE "user".nickname = $1;`,
		f.User).Scan(&id, &f.User);
		err != nil {
		tx.Rollback()
		return nil, &Error{Type: ErrorNotFound, Message: "author not found"}
	}

	if err := tx.QueryRow(`SELECT slug::text, title, posts, threads FROM forum WHERE slug = $1`, f.Slug).
		Scan(&f.Slug, &f.Title, &f.Posts, &f.Threads); err == nil {
		//1
		tx.Rollback()
		return f, &Error{Type: ErrorAlreadyExists, Message: "forum already exists"}
	}

	if err := tx.QueryRow(`INSERT INTO forum (slug, title, owner_id, owner_nickname) VALUES ($1, $2, $3, $4)
									ON CONFLICT DO NOTHING
									RETURNING slug::text, title, posts, threads;`,
		f.Slug, f.Title, id, f.User).Scan(&f.Slug, &f.Title, &f.Posts, &f.Threads);
		err != nil {
		//1
		tx.Rollback()
		return f, &Error{Type: ErrorAlreadyExists, Message: "forum already exists"}
	}

	//fmt.Printf("%+v", *f)
	tx.Commit()
	return f, nil
}

func (f *Forum) GetThreads(ctx *routing.Context) (*Threads, *Error) {
	tx := database.DB
	threads := Threads{}
	var forumId int

	err := tx.QueryRow(`SELECT id FROM forum WHERE slug = $1`, f.Slug).Scan(&forumId)
	//1

	if err == nil {
		rows, err := tx.Query(`SELECT t.id, t.slug::text, t.title, t.message, t.owner_nickname::text as author,
			t.forum_slug::text as forum, t.created, t.votes
		FROM thread t
		WHERE
		t.forum_id = $1 AND
		CASE WHEN $2::timestamp with time zone IS NOT NULL
		THEN
		CASE WHEN $3 = 'DESC'
      	THEN created <= $2
		ELSE created >= $2
		END
		ELSE TRUE
		END
		ORDER BY created `+ ctx.Get("sort").(string) + ` LIMIT ` + ctx.Get("limit").(string) + `;`,
			forumId, ctx.Get("since"), ctx.Get("sort"))

		//1
		//1

		defer rows.Close()

		if err != nil {
			//1
			//tx.Rollback()
		}

		for rows.Next() {
			thr := Thread{}
			err := rows.Scan(&thr.ID, &thr.Slug, &thr.Title, &thr.Message, &thr.Author, &thr.Forum, &thr.Created, &thr.Votes)
			if err != nil {
				//tx.Rollback()
				//1
			}
			//thr.Created, err = time.Parse("2006-01-02T15:04:05.000+03:00", thr.Created.String())
			//if err != nil {
			//	log.Fatal(err)
			//}
			threads = append(threads, &thr)
		}
		//tx.Commit()
		////1
		return &threads, nil

	} else {
		//tx.Rollback()
		return nil, &Error{Message: "forum " + f.Slug + " doesn't exists"}
	}
}

func (f *Forum) UpdateCountThreads() error {
	tx, _ := database.DB.Begin()
	_, err := tx.Exec("UPDATE forum SET threads = threads + 1 WHERE forum.slug = $1", f.Slug)
	if err != nil {
		tx.Rollback()
		return err
	} else {
		tx.Commit()
		return nil
	}
}

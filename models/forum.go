package models

import (
	"github.com/go-ozzo/ozzo-validation"
	"github.com/zwirec/tech-db/db"
	"regexp"
	"fmt"
	"log"
	"github.com/qiangxue/fasthttp-routing"
)

var regexpForumSlug = regexp.MustCompile(`^(\d|\w|-|_)*(\w|-|_)(\d|\w|-|_)*$`)

var forumRuleSlug = []validation.Rule{
	validation.Required,
	validation.Match(regexpForumSlug),
}

//easyjson:json
type Forum struct {
	Posts   int32     `json:"posts"`
	Slug    string    `json:"slug"`
	Threads int32     `json:"threads"`
	Title   string    `json:"title"`
	User    string    `json:"user"`
}

func (f *Forum) Users(ctx *routing.Context) (Users, error) {
	users := Users{}
	tx := database.DB.MustBegin()
	limit := ctx.Get("limit").(string)
	sort := ctx.Get("sort").(string)

	if err := tx.QueryRowx(`SELECT id FROM forum f WHERE f.slug = $1`, ctx.Get("forum_slug")).
				Scan(new (int)); err != nil {
		tx.Rollback()
		return nil, &database.DBError{Type:database.ERROR_DONT_EXISTS, Model: "forum"}
	}

	rows, err := tx.Queryx(`SELECT DISTINCT ON (u.nickname)
						  u.nickname,
						  u.fullname,
						  u.email,
						  u.about
						FROM post p
						  JOIN "user" u ON p.owner_id = u.id
						  JOIN thread t ON p.thread_id = t.id
						  JOIN forum f ON t.forum_id = f.id AND f.slug = $1
						  WHERE CASE WHEN $3 = 'ASC' THEN
						  	u.nickname > $2
						  	ELSE CASE WHEN $2 != '' THEN
								u.nickname < $2
								ELSE TRUE
								END
						  	END
						UNION
						SELECT DISTINCT ON (u.nickname)
						  u.nickname,
						  u.fullname,
						  u.email,
						  u.about
						FROM thread t
						JOIN "user" u ON t.owner_id = u.id
						JOIN forum f ON t.forum_id = f.id AND f.slug = $1
						WHERE CASE WHEN $3 = 'ASC' THEN
							u.nickname > $2
							ELSE CASE WHEN $2 != '' THEN
								u.nickname < $2
								ELSE TRUE
								END
							END
						ORDER BY nickname ` + sort + ` LIMIT ` + limit,
						ctx.Get("forum_slug"),
						ctx.Get("since"),
						sort)
	if err != nil {
		log.Fatal(err)
		tx.Rollback()
		return nil, nil
	}
	defer rows.Close()
	for rows.Next() {
		user := User{}
		rows.StructScan(&user)
		users = append(users, &user)
	}
	tx.Commit()
	return users, nil
}


func (f *Forum) Select() (*Forum, error) {
	dba := database.DB
	tx := dba.MustBegin()
	err := tx.QueryRowx(`SELECT
  slug, posts, threads, title, u.nickname as user FROM forum f, "user" u WHERE f.owner_id = u.id AND lower(f.slug) = lower($1)`, f.Slug).StructScan(f)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "forum"}
	}
	tx.Commit()
	return f, nil
}

func (f *Forum) Validate() error {
	return validation.ValidateStruct(f,
		validation.Field(&f.Slug, forumRuleSlug...),
		validation.Field(&f.Title, validation.Required),
		validation.Field(&f.User, validation.Required),
	)
}

func (f *Forum) Create() (*Forum, error) {
	dba := database.DB
	tx := dba.MustBegin()
	var id int

	if err := tx.QueryRowx(`SELECT id, nickname
									FROM "user"
									WHERE lower("user".nickname) = lower($1);`,
		f.User).Scan(&id, &f.User);
		err != nil {
		tx.Rollback()
		return nil, &database.DBError{Type: database.ERROR_DONT_EXISTS, Model: "user"}
	}

	if err := tx.QueryRowx(`SELECT slug, posts, threads, title FROM forum WHERE lower(slug) = lower($1)`, f.Slug).
		StructScan(f); err == nil {
		tx.Rollback()
		return f, &database.DBError{Type: database.ERROR_ALREADY_EXISTS, Model: "forum"}
	}




	if err := tx.QueryRowx(`INSERT INTO forum (slug, title, owner_id) VALUES ($1, $2, $3)
									ON CONFLICT DO NOTHING
									RETURNING slug, title, posts, threads;`,
		f.Slug, f.Title, id).StructScan(f);
		err != nil {
		log.Println(err)
		tx.Rollback()
		return f, &database.DBError{Model: "forum", Type: database.ERROR_ALREADY_EXISTS}
	}

	//fmt.Printf("%+v", *f)
	tx.Commit()
	return f, nil
}

func (f *Forum) GetThreads(ctx *routing.Context) (*Threads, error) {
	dba := database.DB
	tx := dba.MustBegin()
	threads := Threads{}
	var forum_id int
	err := tx.QueryRowx(`SELECT id FROM forum WHERE lower(slug) = lower($1)`, f.Slug).Scan(&forum_id)
	//log.Println(ctx.Get("sort"))
	if err == nil {
		//log.Println(ctx.Get("since"))
		query := fmt.Sprintf(`SELECT t.id, t.slug, t.title, t.message, u.nickname as author, f.slug as forum, t.created, t.votes FROM thread t
										JOIN "user" u ON (u.id = t.owner_id)
										JOIN forum f ON (f.id = t.forum_id)
										WHERE
										CASE WHEN $2::timestamp with time zone IS NOT NULL
    									THEN
      										CASE WHEN $3 = 'DESC'
        									THEN created <= $2 AND t.forum_id = $1
      										ELSE created >= $2 AND t.forum_id = $1
      										END
      									ELSE t.forum_id = $1
      									END
      									ORDER BY created %s LIMIT %s`,
			ctx.Get("sort"), ctx.Get("limit"))

		rows, err := tx.Queryx(query,
			forum_id, ctx.Get("since"), ctx.Get("sort"))

		defer rows.Close()

		if err != nil {
			log.Println(err)
			tx.Rollback()
		}

		for rows.Next() {
			thread := Thread{}
			err := rows.StructScan(&thread)
			if err != nil {
				tx.Rollback()
				log.Println(err)
			}
			threads = append(threads, &thread)
			//log.Printf("%+v\n", *thread.ID)
		}
		tx.Commit()
		return &threads, nil

	} else {
		tx.Rollback()
		return nil, &database.DBError{Model: "forum", Type: database.ERROR_DONT_EXISTS}
	}
}

func (f *Forum) UpdateCountThreads() error {
	dba := database.DB
	tx := dba.MustBegin()
	_, err := tx.Exec("UPDATE forum SET threads = threads + 1 WHERE forum.slug = $1", f.Slug)
	if err != nil {
		tx.Rollback()
		return err
	} else {
		tx.Commit()
		return nil
	}
}

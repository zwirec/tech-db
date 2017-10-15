package service

import (
	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/models"
	"github.com/mailru/easyjson"
	"log"
	"time"
	"strconv"
	"github.com/zwirec/tech-db/db"
	"github.com/valyala/fasthttp"
	"bytes"
)

func typeHandler(ctx *routing.Context) error {
	ctx.Response.Header.SetContentType("application/json")
	return nil
}

func forumHandler(ctx *routing.Context) error {
	switch ctx.Param("action") {
	case "create":
		if ctx.Param("slug") == "" {
			createForumHandler(ctx)
		} else {
			createThreadHandler(ctx)
		}
	case "details":
		detailsForumHandler(ctx)
	case "threads":
		threadsForumHandler(ctx)
	case "users":
		usersForumHandler(ctx)
	default:
		notFoundResponse(ctx)
		return nil
	}
	return nil
}

func threadHandler(ctx *routing.Context) error {
	log.SetFlags(log.Llongfile)
	if string(ctx.Method()) == "POST" {
		switch ctx.Param("action") {
		case "create":
			createPostHandler(ctx)
		case "details":
			updateThreadHandler(ctx)
		case "vote":
			//log.Println("Voting...")
			voteThreadHandler(ctx)
		}
	} else {
		switch ctx.Param("action") {
		case "details":
			detailsThreadHandler(ctx)
		case "posts":
			//log.Println("Get posts...")
			postsThreadHandler(ctx)
		}
	}
	return nil
}

func userHandler(ctx *routing.Context) error {
	if string(ctx.Method()) == "POST" {
		switch ctx.Param("action") {
		case "create":
			createUserHandler(ctx)
		case "profile":
			//log.Println("update user profile...")
			updateUserHandler(ctx)
		default:
			notFoundResponse(ctx)
			return nil
		}
	} else {
		switch ctx.Param("action") {
		case "profile":
			//log.Println("Get user profile...")
			profileUserHandler(ctx)
		default:
			notFoundResponse(ctx)
			return nil
		}

	}
	return nil
}

func serviceHandler(ctx *routing.Context) error {
	if string(ctx.Method()) == "POST" {
		if ctx.Param("action") == "clear" {
			serviceClearHandler(ctx)
			return nil
		}
		notFoundResponse(ctx)
		return nil
	}
	if ctx.Param("action") == "status" {
		serviceStatusHandler(ctx)
	}
	return nil
}

func postHandler(ctx *routing.Context) error {
	if string(ctx.Method()) == "POST" {
		postUpdateHandler(ctx)
		return nil
	} else {
		postDetailsHandler(ctx)
		return nil
	}
}

func postUpdateHandler(ctx *routing.Context) {
	post := models.Post{}

	if err := easyjson.Unmarshal(ctx.PostBody(), &post); err != nil {
		log.Println(err)
		badRequestResponse(ctx)
	}
	if id, err := strconv.Atoi(ctx.Param("id")); err == nil {
		post.ID = int32(id)
	} else {
		log.Println(err)
	}

	postUpdateResponse(ctx, &post)

}

func postUpdateResponse(ctx *routing.Context, post *models.Post) {
	post, err := post.Update()
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{Message: "post don't exists"}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(post, ctx)
		return
	}
}

func postDetailsHandler(ctx *routing.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))

	post := models.Post{ID: int32(id)}
	related := make(map[string]bool, 4)

	if len(ctx.QueryArgs().PeekMulti("related")) != 0 {
		for _, rel := range bytes.Split(ctx.QueryArgs().PeekMulti("related")[0], []byte(",")) {
			//log.Println(string(rel))
			related[string(rel)] = true
		}
	}


	//log.Printf("%+v\n", related)
	ctx.Set("related", related)
	postDetailsResponse(ctx, &post)
}

func postDetailsResponse(ctx *routing.Context, post *models.Post) {
	postFull, err := post.Details(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{Message: "post don't exists"}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(postFull, ctx)
		return
	}
}

func postsThreadHandler(ctx *routing.Context) {
	checkSlugOrID(ctx)
	limit := ctx.QueryArgs().GetUintOrZero("limit")

	if limit == 0 {
		ctx.Set("limit", "ALL")
	} else {
		ctx.Set("limit", strconv.Itoa(limit))
	}
	//log.Println(string(ctx.QueryArgs().Peek("desc")))

	if ctx.QueryArgs().GetBool("desc") == false {
		ctx.Set("sort", "ASC")
	} else {
		ctx.Set("sort", "DESC")
	}

	if ctx.QueryArgs().GetUintOrZero("since") == 0 {
		ctx.Set("since", -1)
	} else {
		ctx.Set("since", ctx.QueryArgs().GetUintOrZero("since"))
	}

	ctx.Set("sort_type", string(ctx.QueryArgs().Peek("sort")))
	thread := models.Thread{}
	postsThreadResponse(ctx, &thread)
}

func postsThreadResponse(ctx *routing.Context, thread *models.Thread) {
	posts, err := thread.Posts(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{Message: "thread don't exists"}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(posts, ctx)
		return
	}
}

func voteThreadHandler(ctx *routing.Context) {
	checkSlugOrID(ctx)
	voice := models.Vote{}
	if err := easyjson.Unmarshal(ctx.PostBody(), &voice); err != nil {
		log.Println(err)
		return
	}
	voteThreadResponse(ctx, &voice)
}

func voteThreadResponse(ctx *routing.Context, voice *models.Vote) {
	thread, err := voice.Vote(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{Message: "thread don't exists"}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(thread, ctx)
		return
	}
}

func serviceStatusHandler(ctx *routing.Context) {
	status := models.Status{}
	serviceStatusResponse(ctx, &status)
}

func serviceStatusResponse(ctx *routing.Context, status *models.Status) {
	status, err := status.Status()
	if err != nil {
		log.Println(err)
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(status, ctx)
		return
	}
}

func serviceClearHandler(ctx *routing.Context) {
	serviceClearResponse(ctx)
}

func serviceClearResponse(ctx *routing.Context) {
	if err := models.Clear(); err != nil {
		ctx.SetStatusCode(fasthttp.StatusOK)
		return
	}
}

func detailsThreadHandler(ctx *routing.Context) {
	thread := models.Thread{}
	checkSlugOrID(ctx)
	detailsThreadResponse(ctx, &thread)
}

func detailsThreadResponse(ctx *routing.Context, thread *models.Thread) {
	thread, err := thread.Details(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(thread, ctx)
		return
	}
}

func updateThreadHandler(ctx *routing.Context) {
	threadUpdate := models.ThreadUpdate{}
	checkSlugOrID(ctx)
	if err := easyjson.Unmarshal(ctx.PostBody(), &threadUpdate); err != nil {
		log.Fatal(err)
	}
	updateThreadResponse(ctx, &threadUpdate)
}

func updateThreadResponse(ctx *routing.Context, threadUpdate *models.ThreadUpdate) {
	thread, err := threadUpdate.Update(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(thread, ctx)
		return
	}
}

func createForumHandler(ctx *routing.Context) error {
	//log.Println("Create forum...")
	forum := models.Forum{}
	if err := easyjson.Unmarshal(ctx.PostBody(), &forum); err != nil {
		log.Println(err)
		return err
	}
	if err := forum.Validate(); err != nil {
		log.Println(err)
		badRequestResponse(ctx)
		return nil
	}
	createForumResponse(ctx, &forum)
	return nil
}

func createThreadHandler(ctx *routing.Context) error {
	//log.Println("Create thread...")
	thread := models.Thread{Created: time.Now()}
	if err := easyjson.Unmarshal(ctx.PostBody(), &thread); err != nil {
		log.Fatal(err)
	}
	thread.Forum = ctx.Param("slug")
	//if err := thread.Validate(); err != nil {
	//	log.Println(err)
	//	badRequestResponse(ctx)
	//	return nil
	//}

	createThreadResponse(ctx, &thread)
	return nil
}

func createPostHandler(ctx *routing.Context) error {
	//log.Println("Create post...")
	checkSlugOrID(ctx)
	posts := models.Posts{}
	if err := easyjson.Unmarshal(ctx.PostBody(), &posts); err != nil {
		log.Println(err)
	}
	ctx.Set("created", time.Now().Format(time.RFC3339Nano))
	//for _, post := range posts {
	//	if err := post.Validate(); err != nil {
	//		log.Println(err)
	//		return err
	//	}
	//}
	createPostResponse(ctx, &posts)
	return nil
}

func checkSlugOrID(ctx *routing.Context) {
	thread_slug := ctx.Param("slug_or_id")
	thread_id, err := strconv.Atoi(thread_slug)
	if err != nil {
		ctx.Set("thread_slug", thread_slug)
	} else {
		ctx.Set("thread_id", thread_id)
	}
}

func createUserHandler(ctx *routing.Context) error {
	//log.Println("Create user...")
	nickname := ctx.Param("nickname")
	user := models.User{}
	if err := easyjson.Unmarshal(ctx.PostBody(), &user); err != nil {
		log.Println(err)
		return err
	}
	user.Nickname = nickname
	if err := user.Validate(); err != nil {
		log.Println(err)
		badRequestResponse(ctx)
		return nil
	}
	createUserResponse(ctx, &user)
	return nil
}

func detailsForumHandler(ctx *routing.Context) error {
	forum := models.Forum{Slug: ctx.Param("slug")}
	detailsForumResponse(ctx, &forum)
	return nil
}

func threadsForumHandler(ctx *routing.Context) error {
	forum := models.Forum{Slug: ctx.Param("slug")}

	limit := ctx.QueryArgs().GetUintOrZero("limit")

	if limit == 0 {
		ctx.Set("limit", "ALL")
	} else {
		ctx.Set("limit", strconv.Itoa(limit))
	}

	//log.Println(ctx.Get("limit"))

	//log.Println(string(ctx.QueryArgs().QueryString()))

	if ctx.QueryArgs().GetBool("desc") == false {
		ctx.Set("sort", "ASC")
	} else {
		ctx.Set("sort", "DESC")
	}

	//log.Println(ctx.Get("sort"))

	//log.Println(string(ctx.QueryArgs().QueryString()))

	_, err := time.Parse("2006-01-02T15:04:05Z", string(ctx.QueryArgs().Peek("since")))

	if err == nil {
		ctx.Set("since", string(ctx.QueryArgs().Peek("since")))
	}

	threadsForumResponse(ctx, &forum)
	return nil
}

func usersForumHandler(ctx *routing.Context) error {
	forum := models.Forum{}

	ctx.Set("forum_slug", ctx.Param("slug"))

	limit := ctx.QueryArgs().GetUintOrZero("limit")

	if limit == 0 {
		ctx.Set("limit", "ALL")
	} else {
		ctx.Set("limit", strconv.Itoa(limit))
	}

	if ctx.QueryArgs().GetBool("desc") == false {
		ctx.Set("sort", "ASC")
	} else {
		ctx.Set("sort", "DESC")
	}

	if string(ctx.QueryArgs().Peek("since")) != "" {
		ctx.Set("since", string(ctx.QueryArgs().Peek("since")))
	} else {
		ctx.Set("since", "")
	}

	usersForumResponse(ctx, &forum)
	return nil
}

func usersForumResponse(ctx *routing.Context, forum *models.Forum) {
	users, err := forum.Users(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			easyjson.MarshalToWriter(&models.Error{}, ctx)
			return
		case database.ERROR_ALREADY_EXISTS:
			ctx.SetStatusCode(fasthttp.StatusConflict)
			easyjson.MarshalToWriter(&models.Error{}, ctx)
			return
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(users, ctx)
		return
	}
}

func updateUserHandler(ctx *routing.Context) {
	userUpd := models.UserUpdate{}
	if err := easyjson.Unmarshal(ctx.PostBody(), &userUpd); err != nil {
		log.Fatal(err)
	}
	updateUserResponse(ctx, &userUpd)
}

func updateUserResponse(ctx *routing.Context, update *models.UserUpdate) {
	profile, err := update.UpdateProfile(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			easyjson.MarshalToWriter(&models.User{}, ctx)
			return
		case database.ERROR_ALREADY_EXISTS:
			ctx.SetStatusCode(fasthttp.StatusConflict)
			easyjson.MarshalToWriter(&models.User{}, ctx)
			return
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(profile, ctx)
		return
	}

}

func profileUserHandler(ctx *routing.Context) {
	user := models.User{Nickname: ctx.Param("nickname")}
	profileUserResponse(ctx, &user)
}

func profileUserResponse(ctx *routing.Context, user *models.User) {
	profile, err := user.GetProfile()
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(profile, ctx)
		return
	}
}

func createThreadResponse(ctx *routing.Context, thread *models.Thread) {
	thread, err := thread.Create()
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_ALREADY_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusConflict)
				easyjson.MarshalToWriter(thread, ctx)
				return
			}
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusCreated)
		easyjson.MarshalToWriter(thread, ctx)
		return
	}
}

func createPostResponse(ctx *routing.Context, posts *models.Posts) {
	posts, err := posts.Create(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			ctx.SetStatusCode(fasthttp.StatusConflict)
			easyjson.MarshalToWriter(&models.Error{}, ctx)
			return
		case database.ERROR_ALREADY_EXISTS:
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			easyjson.MarshalToWriter(&models.Error{}, ctx)
			return
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusCreated)
		easyjson.MarshalToWriter(posts, ctx)
		return
	}
}

func createUserResponse(ctx *routing.Context, user *models.User) {
	users, err := user.Create()
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_ALREADY_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusConflict)
				easyjson.MarshalToWriter(*users, ctx)
				return
			}
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusCreated)
		if len(*users) == 1 {
			easyjson.MarshalToWriter((*users)[0], ctx)
		} else {
			easyjson.MarshalToWriter(*users, ctx)
		}
		return
	}
}

func createForumResponse(ctx *routing.Context, forum *models.Forum) {
	forum, err := forum.Create()
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_ALREADY_EXISTS:
			{
				//log.Println("forum already exists")
				ctx.SetStatusCode(fasthttp.StatusConflict)
				easyjson.MarshalToWriter(forum, ctx)
				return
			}
		case database.ERROR_DONT_EXISTS:
			{
				//log.Println("user don't exists")
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusCreated)
		easyjson.MarshalToWriter(forum, ctx)
		return
	}
}

func threadsForumResponse(ctx *routing.Context, forum *models.Forum) {
	threads, err := forum.GetThreads(ctx)
	if err != nil {
		switch err.(*database.DBError).Type {
		case database.ERROR_DONT_EXISTS:
			{
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				easyjson.MarshalToWriter(&models.Error{}, ctx)
				return
			}
		}
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(threads, ctx)
		return
	}
}

func detailsForumResponse(ctx *routing.Context, forum *models.Forum) {
	forum, err := forum.Select()
	if err != nil && err.(*database.DBError).Type == database.ERROR_DONT_EXISTS {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		easyjson.MarshalToWriter(&models.Error{}, ctx)
		return
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		easyjson.MarshalToWriter(forum, ctx)
		return
	}
}

func badRequestResponse(ctx *routing.Context) {
	ctx.SetStatusCode(fasthttp.StatusBadRequest)
	return
}

func notFoundResponse(ctx *routing.Context) {
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	return
}

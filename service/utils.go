package service

import (
	"github.com/qiangxue/fasthttp-routing"
	"strconv"
)

func checkSlugOrID(ctx *routing.Context) {
	threadSlug := ctx.Param("slug_or_id")
	threadId, err := strconv.Atoi(threadSlug)
	if err != nil {
		ctx.Set("thread_slug", threadSlug)
	} else {
		ctx.Set("thread_id", threadId)
	}
}

func typeHandler(ctx *routing.Context) error {
	ctx.Response.Header.SetContentType("application/json")
	return nil
}

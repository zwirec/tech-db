package service

import (
	"github.com/valyala/fasthttp"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/zwirec/tech-db/models"
	"log"
	"github.com/mailru/easyjson"
	_ "github.com/mkevac/debugcharts"
)

type Service struct {
	server *fasthttp.Server
	router *routing.Router
}

func NewService() *Service {
	return &Service{server: &fasthttp.Server{}, router: routing.New()}
}

func (svc *Service) Run() {
	svc.router.Get("/", func(context *routing.Context) error {
		user := models.User{"jack", "jack.sparrow", "jack.sparrow@mail.ru", "it's okay"}
		if err := user.Validate(); err != nil {
			log.Fatal(err)
		}
		easyjson.MarshalToWriter(user, context)
		return nil
	})
	svc.server.Handler = svc.router.HandleRequest
	svc.server.ListenAndServe("127.0.0.1:10000")
}

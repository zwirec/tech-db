package service

import (
	"log"

	"runtime"

	_ "github.com/mkevac/debugcharts"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
	"github.com/zwirec/tech-db/db"
	"io/ioutil"
)

type Service struct {
	server *fasthttp.Server
	router *routing.Router
}

func NewService() *Service {
	return &Service{server: &fasthttp.Server{}, router: routing.New()}
}

func (svc *Service) Run() error {

	runtime.GOMAXPROCS(runtime.NumCPU())

	svc.setEndpointHandlers()

	log.SetFlags(log.Llongfile | log.Ltime)

	log.SetOutput(ioutil.Discard)
	if err := database.InitDB(); err != nil {
		log.Fatal(err)
	}
	svc.server.Handler = svc.router.HandleRequest
	svc.server.MaxConnsPerIP = 10000
	svc.server.Concurrency = 10000
	return svc.server.ListenAndServe(":5000")
}

func (svc *Service) setEndpointHandlers() {
	svc.router.To("GET,POST", "/api/forum/<slug>/<action>", forumHandler, typeHandler)
	svc.router.Post("/api/forum/<action>", forumHandler, typeHandler)
	svc.router.To("GET,POST", "/api/user/<nickname>/<action>", userHandler, typeHandler)
	svc.router.To("GET,POST", "/api/thread/<slug_or_id>/<action>", threadHandler, typeHandler)
	svc.router.To("GET,POST", "/api/service/<action>", serviceHandler, typeHandler)
	svc.router.To("GET,POST", "/api/post/<id>/details", postHandler, typeHandler)
}

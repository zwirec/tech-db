package main

import (
	"github.com/zwirec/tech-db/service"
	_ "net/http/pprof"
	"net/http"
	"sync"
)

func main() {
	service := service.NewService()
	go http.ListenAndServe("0.0.0.0:1111", nil)
	var wg sync.WaitGroup
	wg.Add(1)
	go service.Run()
	wg.Wait()
}

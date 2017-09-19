package main

import (
	"net/http"
	_ "net/http/pprof"
	"time"
)

func allocAndKeep() {
	var b [][]byte
	for {
		b = append(b, make([]byte, 1024))
		time.Sleep(time.Millisecond)
	}
}

func allocAndLeave() {
	var b [][]byte
	for {
		b = append(b, make([]byte, 1024))
		if len(b) == 20 {
			b = nil
		}
		time.Sleep(time.Millisecond)
	}
}

func main() {
	go allocAndKeep()
	go allocAndLeave()
	http.ListenAndServe("0.0.0.0:8080", nil)
}

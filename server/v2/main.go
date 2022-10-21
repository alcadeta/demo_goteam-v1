package main

import (
	handlers2 "github.com/kxplxn/goteam/server/v2/handlers"
	"github.com/kxplxn/goteam/server/v2/relay"
	"log"
	"net/http"
)

func main() {
	if err := runWebAPI(); err != nil {
		log.Fatal(err)
	}
}

func runWebAPI() error {
	logger := relay.NewAPILogger()

	return handleRoutes(map[string]http.Handler{
		"/":         handlers2.NewRoot(logger),
		"/register": handlers2.NewRegister(logger),
	}, ":1337")
}

func handleRoutes(routeHandlers map[string]http.Handler, port string) error {
	mux := http.NewServeMux()
	for route, handler := range routeHandlers {
		mux.Handle(route, handler)
	}
	return http.ListenAndServe(port, mux)
}

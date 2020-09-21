package main

import (
	"fmt"
	"net/http"
	"speech-to-text-back/src/server"
)

func main() {
	server.Upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	s := http.Server{
		Addr:    ":8080",
		Handler: new(server.Handler),
	}

	// start listening
	fmt.Println("Started server on port 8080")
	err := s.ListenAndServe()

	if err == nil {
		panic("Something went wrong with the server")
	}
}

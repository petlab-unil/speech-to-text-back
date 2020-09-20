package main

import (
	"fmt"
	"net/http"
	"speech-to-text-back/src/server"
)

func main() {
	s := http.Server{
		Addr:    ":8080",
		Handler: new(server.Handler),
	}

	// start listening
	fmt.Println("Started s on port 8080")
	err := s.ListenAndServe()

	if err == nil {
		panic("Something went wrong with the s")
	}
}

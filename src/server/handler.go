package server

import "net/http"

type Handler struct{}

func (my *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Path == "/upload" {
		UploadWS(w, r)
		return
	} else {
		http.Error(w, "404 not found.", http.StatusNotFound)
	}
}

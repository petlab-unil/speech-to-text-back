package server

import (
	"gopkg.in/mgo.v2"
	"log"
	"net/http"
	"os"
)

type Handler struct {
	MongoSession *mgo.Session
	routes       RouteTree
}

func NewHandler() *Handler {
	h := new(Handler)
	h.defineRoutes()
	session, err := mgo.Dial(os.Getenv("MONGO_HOSTNAME"))
	println(os.Getenv("MONGO_HOSTNAME"))
	if err != nil {
		log.Fatal(err.Error())
	}

	h.MongoSession = session

	return h
}

func (h *Handler) defineRoutes() {
	h.routes.path = "/"
	h.routes.children = make(map[string]*RouteTree)
	h.routes.RegisterRoute("/account/login", Login)
	h.routes.RegisterRoute("/account/create", AccountCreate)
	h.routes.RegisterRoute("/sessions/check", SessionsCheck)
	h.routes.RegisterRoute("/me", MyAccount)
	h.routes.RegisterRoute("/upload", UploadWS)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.routes.ExecuteQuery(h, w, r)
}

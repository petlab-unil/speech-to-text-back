package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"io"
	"log"
	"net/http"
	"speech-to-text-back/src/server/account"
	"strconv"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  32000,
	WriteBufferSize: 1024,
}

func SessionsCheck(_ *Handler, w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w, "Ok")
}

func MyAccount(h *Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	auth := r.Header.Get("Authorization")

	a, err := account.FullAccount(h.MongoSession, auth)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if a == nil {
		http.Error(w, "Did not find account", http.StatusBadRequest)
		return
	}

	serialized, err := json.Marshal(a)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, _ = fmt.Fprintf(w, string(serialized))
}

func AccountCreate(h *Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	var a account.Account
	err := json.NewDecoder(r.Body).Decode(&a)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = account.CreateAccount(&a, h.MongoSession)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprintf(w, "Account created")
}

func Login(h *Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	var a account.Account
	err := json.NewDecoder(r.Body).Decode(&a)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := account.IdentifyAccount(&a, h.MongoSession)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if id == nil {
		http.Error(w, "Invalid password", http.StatusForbidden)
		return
	}

	session, err := account.CreateSession(*id, h.MongoSession)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if session == nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	serialized, _ := json.Marshal(session)
	_, _ = io.WriteString(w, string(serialized))
}

func UploadWS(h *Handler, w http.ResponseWriter, r *http.Request) {
	audioTypeStr := r.URL.Query().Get("audioType")
	audioTypeInt, err := strconv.Atoi(audioTypeStr)
	audioType := speechpb.RecognitionConfig_AudioEncoding(audioTypeInt)
	if err != nil || audioType < 0 || audioType > 7 {
		http.Error(w, "Invalid, missing or malformed audioType", http.StatusNotFound)
		return
	}

	sizeStr := r.URL.Query().Get("size")
	sizeInt, err := strconv.Atoi(sizeStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query param for size: %s", sizeStr), http.StatusNotFound)
		return
	}

	fileName := r.URL.Query().Get("name")

	if len(fileName) == 0 {
		http.Error(w, "Missing or malformed file name", http.StatusNotFound)
		return
	}

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	auth := r.URL.Query().Get("Authorization")

	session, err := account.FindSession(h.MongoSession, auth)

	if err != nil {
		log.Println(err)
		return
	}

	newTranslation, err := account.CreateTranslation(h.MongoSession, fileName, session.User)

	if err != nil {
		log.Println(err)
		return
	}

	streamS2t(h, conn, sizeInt, newTranslation, audioType)
}

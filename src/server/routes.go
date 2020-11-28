package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
	"net/http"
	"speech-to-text-back/src/server/account"
	"strconv"
)

func SessionsCheck(_ *Handler, w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w, "Ok")
}

func MyAccount(h *Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	auth := r.Header.Get("Authorization")

	sessionCopy := h.MongoSession.Copy()
	defer sessionCopy.Close()

	a, err := account.FullAccount(sessionCopy, auth)

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

func AccountList(h *Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	sessionCopy := h.MongoSession.Clone()
	defer sessionCopy.Close()

	accounts, err := account.AllAccounts(sessionCopy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serialized, err := json.Marshal(accounts)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, _ = fmt.Fprintf(w, string(serialized))
}

func OneTranslation(h *Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	auth := r.Header.Get("Authorization")

	sessionCopy := h.MongoSession.Copy()
	defer sessionCopy.Close()
	sessOid := bson.ObjectIdHex(auth)
	collection := sessionCopy.DB("s2t").C("sessions")
	var sess account.Session
	err := collection.FindId(sessOid).One(&sess)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection = sessionCopy.DB("s2t").C("translations")

	queryId := r.URL.Query()["id"][0]

	var t account.Translation
	err = collection.FindId(bson.ObjectIdHex(queryId)).One(&t)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serialized, err := json.Marshal(t)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, _ = fmt.Fprintf(w, string(serialized))
}

type TranslationShareRequest struct {
	TranslationId  string `json:"translationId"`
	AccountToShare string `json:"accountToShare"`
}

func TranslationShare(h *Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	var a TranslationShareRequest
	err := json.NewDecoder(r.Body).Decode(&a)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sessionCopy := h.MongoSession.Copy()
	defer sessionCopy.Close()

	err = account.ShareTranslation(sessionCopy, &a.TranslationId, &a.AccountToShare)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, _ = fmt.Fprintf(w, "ok")
}

func TranslationDelete(h *Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	auth := r.Header.Get("Authorization")

	sessionCopy := h.MongoSession.Copy()
	defer sessionCopy.Close()
	sessOid := bson.ObjectIdHex(auth)
	collection := sessionCopy.DB("s2t").C("sessions")
	var sess account.Session
	err := collection.FindId(sessOid).One(&sess)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	queryId := r.URL.Query()["id"][0]

	err = account.DeleteTranslation(sessionCopy, &sess, &queryId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, _ = fmt.Fprintf(w, "ok")
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

	sessionCopy := h.MongoSession.Copy()
	defer sessionCopy.Close()
	err = account.CreateAccount(&a, sessionCopy)

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

	sessionCopy := h.MongoSession.Copy()
	defer sessionCopy.Close()

	id, err := account.IdentifyAccount(&a, sessionCopy)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if id == nil {
		http.Error(w, "Invalid password", http.StatusForbidden)
		return
	}

	session, err := account.CreateSession(*id, sessionCopy)

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
	packetSize := r.URL.Query().Get("packetSize")
	sampleRateHertzStr := r.URL.Query().Get("sampleRateHertz")
	sizeInt, err := strconv.Atoi(sizeStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query param for size: %s", sizeStr), http.StatusNotFound)
		return
	}
	packetInt, err := strconv.Atoi(packetSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query param for size: %s", packetSize), http.StatusNotFound)
		return
	}

	sampleRateHertz, err := strconv.Atoi(sampleRateHertzStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query param for size: %s", sampleRateHertzStr), http.StatusNotFound)
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  packetInt,
		WriteBufferSize: 1024,
	}

	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	auth := r.URL.Query().Get("Authorization")

	sessionCopy := h.MongoSession.Copy()
	defer sessionCopy.Close()
	session, err := account.FindSession(sessionCopy, auth)

	if err != nil {
		log.Println(err)
		return
	}

	fileName := r.URL.Query().Get("name")

	if len(fileName) == 0 {
		http.Error(w, "Missing or malformed file name", http.StatusNotFound)
		return
	}

	newTranslation, err := account.CreateTranslation(sessionCopy, fileName, session.User)

	if err != nil {
		log.Println(err)
		return
	}

	model := r.URL.Query().Get("model")
	language := r.URL.Query().Get("language")

	streamS2t(h, conn, sizeInt, newTranslation, packetInt, sampleRateHertz, audioType, language, model)
}

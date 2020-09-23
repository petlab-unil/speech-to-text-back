package account

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"speech-to-text-back/src/server"
)

func Create(h *server.Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	var account Account
	err := json.NewDecoder(r.Body).Decode(&account)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = createAccount(&account, h.MongoSession)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, _ = fmt.Fprintf(w, "Account created")
}

func Login(h *server.Handler, w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	var account Account
	err := json.NewDecoder(r.Body).Decode(&account)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	checked, err := identifyAccount(&account, h.MongoSession)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !checked {
		http.Error(w, "Invalid password", http.StatusForbidden)
		return
	}

	session, err := createSession(&account, h.MongoSession)

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

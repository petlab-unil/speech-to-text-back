package server

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
)

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  32000,
	WriteBufferSize: 1024,
}

func UploadWS(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	sizeInt, err := strconv.Atoi(sizeStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid query param for size: %s", sizeStr), http.StatusNotFound)
		return
	}

	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	streamS2t(conn, sizeInt)
}

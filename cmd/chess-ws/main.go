package main

import (
	"net/http"

	"github.com/nemo984/gochess-websockets/pkg/chess"
)

func main() {
	hub := chess.NewHub()
	go hub.Run()
	http.Handle("/", http.FileServer(http.Dir("./client")))
	http.HandleFunc("/ws", chess.NewWSHandler(hub))
	http.ListenAndServe(":8080", nil)
}

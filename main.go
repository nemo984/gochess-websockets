package main

import (
	"net/http"
)

func main() {
	go hub.run()
	http.Handle("/", http.FileServer(http.Dir("./client")))
	http.HandleFunc("/ws", wsHandler)
	http.ListenAndServe(":8080", nil)
}

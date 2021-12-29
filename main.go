package main

import (
	"net/http"
)

func main() {
	go hub.run()
	http.HandleFunc("/", wsHandler)
	http.ListenAndServe(":8080", nil)
}
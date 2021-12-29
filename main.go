package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}


func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, err := upgrader.Upgrade(w,r, nil)
		if err != nil {
			log.Print("upgrade failed: ", err)
			return
		}
		defer conn.Close()
		fmt.Fprintf(w, "Setting up the server!")

		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read failed: ",err)
				break
			}
			fmt.Println(mt,string(message))
		}
	})

	http.ListenAndServe(":8080", nil)
}
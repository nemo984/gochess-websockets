package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Conn struct {
	ws   *websocket.Conn
	send chan interface{}
}

type Data struct {
	GameID    string    `json:"id,omitempty"`
	Move      string    `json:"move,omitempty"`
	SDP       SDP       `json:"sdp,omitempty"`
	Candidate Candidate `json:"candidate,omitempty"`
}

type Candidate struct {
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdpMid"`
	SDPMLineIndex int    `json:"sdpMLineIndex"`
}

type SDP struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
}

type Message struct {
	Action string `json:"action"`
	Data   Data   `json:"data,omitempty"`
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Conn) readPump(done chan struct{}) {
	defer func() {
		log.Println(c, "socket is closing")
		hub.unregister <- c
		c.ws.Close()
	}()
	defer close(done)
	for {
		m := &Message{}
		err := c.ws.ReadJSON(m)
		log.Printf("Received %#v\n", m)
		if err != nil {
			log.Printf("error: %v\n", err)
			break
		}
		switch strings.ToLower(m.Action) {
		case "create":
			hub.create <- CreateRequest{Conn: c, SDP: m.Data.SDP}
		case "join":
			hub.join <- JoinRequest{Conn: c, GameID: m.Data.GameID, SDP: m.Data.SDP}
		case "move":
			hub.move <- MoveRequest{Conn: c, move: m.Data.Move}
		case "answer":
			hub.answer <- AnswerRequest{Conn: c, SDP: m.Data.SDP}
		case "ice":
			hub.ice <- IceRequest{Conn: c, Candidate: m.Data.Candidate}
		}

	}
}

// write writes a message with the given message type and payload.
func (c *Conn) write(mt int, payload []byte) error {
	return c.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Conn) writePump(done chan struct{}) {
	defer c.ws.Close()
	for {
		select {
		case <-done:
			return
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.ws.WriteJSON(message); err != nil {
				return
			}
		}
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Trying to upgrade")
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	conn := &Conn{
		send: make(chan interface{}),
		ws:   ws,
	}

	done := make(chan struct{})
	go conn.readPump(done)
	go conn.writePump(done)
	hub.register <- conn
}

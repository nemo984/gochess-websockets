package main

import (
	"log"

	"github.com/notnil/chess"
)

type Game struct {
	id string
	game *chess.Game
	white *Conn
	black *Conn
	ongoing bool
}

//Hub maintains set of ongoing games
type Hub struct {
	games map[string]*Game
	gameConnections map[*Conn]string
	register chan *Conn 
	unregister chan *Conn 
	move chan Move
}

var hub = Hub{
	games: make(map[string]*Game),
	gameConnections: make(map[*Conn]string),
	register: make(chan *Conn),
	unregister: make(chan *Conn),
	move: make(chan Move),
}

type Move struct {
	Conn *Conn
	m	string
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			log.Println("New connection: ", conn)

		case conn := <-h.unregister:
			if g,ok := h.gameConnections[conn]; ok {
				delete(h.gameConnections, conn)
				close(conn.send)
				h.games[g].ongoing = false
			}

		case move := <-h.move:
			log.Println(move.Conn, "Plays", move.m)
			if game, ok := h.gameConnections[move.Conn]; ok {
				log.Println(move.Conn, "Plays", move.m, game)
			}
		}
	}
}
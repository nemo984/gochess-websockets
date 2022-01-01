package main

import (
	"log"
)

//Hub maintains set of ongoing games/ events
type Hub struct {
	games      *Games
	register   chan *Conn
	unregister chan *Conn
	create     chan *Conn
	join       chan Join
	move       chan Move
}

var hub = Hub{
	games: &Games{
		games: make(map[string]*Game),
		gameConnections: make(map[*Conn]string),
	},
	register:   make(chan *Conn),
	unregister: make(chan *Conn),
	create:     make(chan *Conn),
	move:       make(chan Move),
	join:       make(chan Join),
}

func (h *Hub) run() {
	log.Println("Hub is Listening")
	defer log.Println("Hub is dead")
	//TODO: resign, draw
	for {
		select {
		case conn := <-h.register:
			log.Println("New connection: ", conn)

		case conn := <-h.create:
			hub.games.create(conn)

		case join := <-h.join:
			hub.games.join(join)

		case conn := <-h.unregister:
			hub.games.leave(conn)

		case move := <-h.move:
			hub.games.move(move)
		}
	}
}

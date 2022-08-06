package main

import (
	"log"
)

//Hub maintains set of ongoing games/ events
type Hub struct {
	gameService *GameService
	register    chan *Conn
	unregister  chan *Conn
	create      chan *Conn
	join        chan JoinRequest
	move        chan MoveRequest
}

var hub = Hub{
	gameService: NewGameService(),
	register:    make(chan *Conn),
	unregister:  make(chan *Conn),
	create:      make(chan *Conn),
	move:        make(chan MoveRequest),
	join:        make(chan JoinRequest),
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
			hub.gameService.create(conn)

		case join := <-h.join:
			hub.gameService.join(join)

		case conn := <-h.unregister:
			hub.gameService.leave(conn)

		case move := <-h.move:
			hub.gameService.move(move)
		}
	}
}

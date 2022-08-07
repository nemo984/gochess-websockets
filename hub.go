package main

import (
	"log"
)

//Hub maintains set of ongoing games/ events
type Hub struct {
	gameService *GameService
	register    chan *Conn
	unregister  chan *Conn
	create      chan CreateRequest
	join        chan JoinRequest
	move        chan MoveRequest
	answer      chan AnswerRequest
}

var hub = Hub{
	gameService: NewGameService(),
	register:    make(chan *Conn),
	unregister:  make(chan *Conn),
	create:      make(chan CreateRequest),
	move:        make(chan MoveRequest),
	join:        make(chan JoinRequest),
	answer:      make(chan AnswerRequest),
}

func (h *Hub) run() {
	log.Println("Hub is Listening")
	defer log.Println("Hub is dead")
	//TODO: resign, draw
	for {
		select {
		case conn := <-h.register:
			log.Println("New connection: ", conn)

		case create := <-h.create:
			hub.gameService.create(create)

		case join := <-h.join:
			log.Println("trying to join")
			hub.gameService.join(join)

		case conn := <-h.unregister:
			hub.gameService.leave(conn)

		case move := <-h.move:
			hub.gameService.move(move)

		case answer := <-h.answer:
			hub.gameService.answer(answer)
		default:
			log.Println("wtf")
		}
	}
}

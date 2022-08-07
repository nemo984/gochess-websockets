package chess

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
	ice         chan IceRequest
}

func NewHub() *Hub {
	return &Hub{
		gameService: NewGameService(),
		register:    make(chan *Conn),
		unregister:  make(chan *Conn),
		create:      make(chan CreateRequest),
		move:        make(chan MoveRequest),
		join:        make(chan JoinRequest),
		answer:      make(chan AnswerRequest),
		ice:         make(chan IceRequest),
	}
}

func (h *Hub) Run() {
	log.Println("Hub is Listening")
	defer log.Println("Hub is dead")
	//TODO: resign, draw
	for {
		select {
		case conn := <-h.register:
			log.Println("New connection: ", conn)

		case create := <-h.create:
			h.gameService.create(create)

		case join := <-h.join:
			log.Println("trying to join")
			h.gameService.join(join)

		case conn := <-h.unregister:
			log.Println("<-h.unregister ", conn)
			//hub.gameService.leave(conn) #TODO: fix lock bug

		case move := <-h.move:
			h.gameService.move(move)

		case answer := <-h.answer:
			h.gameService.answer(answer)

		case ice := <-h.ice:
			h.gameService.ice(ice)
		}
	}
}

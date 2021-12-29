package main

import (
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/notnil/chess"
)

type Game struct {
	id string
	game *chess.Game
	white *Conn
	black *Conn
	ongoing bool
}

//Hub maintains set of ongoing games/ events
type Hub struct {
	games map[string]*Game
	gameConnections map[*Conn]string
	register chan *Conn 
	unregister chan *Conn 
	create chan *Conn
	join chan Join
	move chan Move
}

var hub = Hub{
	games: make(map[string]*Game),
	gameConnections: make(map[*Conn]string),
	register: make(chan *Conn),
	unregister: make(chan *Conn),
	create: make(chan *Conn),
	move: make(chan Move),
	join: make(chan Join),
}

type Move struct {
	Conn *Conn
	move	string
}

type Join struct {
	Conn *Conn
	GameID string
}

type Response struct {
	GameID string `json:"id"`
	FEN	string `json:"fen"`
	PGN string `json:"pgn"`
} 

type ErrResponse struct {
	Message string `json:"message"`
}

func (h *Hub) run() {
	log.Println("Hub is Listening")
	defer log.Println("Hub is dead")
	for {
		select {
		case conn := <-h.register:
			log.Println("New connection: ", conn)
		
		case conn := <-h.create:
			log.Println("New Game Created By: ", conn)
			gameID := RandStringRunes(5)
			log.Println("Game ID:",gameID)
			h.games[gameID] = &Game{
				id: gameID,
				game: chess.NewGame(),
				white: conn, //TODO: optionally create with color
				ongoing: true,
			}
			h.gameConnections[conn] = gameID
			//return Game ID to conn
			conn.send <- Response{
				GameID: gameID,
				FEN: h.games[gameID].game.FEN(),
				PGN: strings.TrimSpace(h.games[gameID].game.String()),
			}

		case join := <-h.join:
			conn := join.Conn
			log.Println(conn,"trying to join", join.GameID)
			if g,ok := h.games[join.GameID]; ok {
				if g.black != nil && g.white != nil {
					conn.err <- ErrResponse{Message: "Game already fulled"}
				}
				if g.white == nil {
					g.white = conn
				} else {
					g.black = conn
				}
				log.Println("Join Game:",join.GameID,"Success!")
				//write back state of the board
				h.gameConnections[conn] = join.GameID
				//broadcast to other guy, this guy joins
				
			}


		case conn := <-h.unregister:
			log.Println(conn,"unregister")

		case m := <-h.move:
			log.Println(m.Conn, "Plays", m.move)
			if game, ok := h.gameConnections[m.Conn]; ok {
				log.Println(m.Conn, "Plays", m.move, game)
			}
		}
	}
}


var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
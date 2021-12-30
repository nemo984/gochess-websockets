package main

import (
	"fmt"
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

//whatever you want to send back
type Response struct {
	GameID string `json:"id"`
	Event string `json:"event"`
	FEN	string `json:"fen"`
	PGN string `json:"pgn"`
} 

//Response in case of error
type ErrResponse struct {
	Message string `json:"message"`
}

func (h *Hub) run() {
	log.Println("Hub is Listening")
	defer log.Println("Hub is dead")
	//TODO: put the cases in their own func/file
	//TODO: resign, draw 
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
			conn.send <- newResponse(h.games[gameID].game, gameID, "Game Created")

		case join := <-h.join:
			conn := join.Conn
			log.Println(conn,"trying to join", join.GameID)
			if g,ok := h.games[join.GameID]; ok {
				if g.black != nil && g.white != nil {
					conn.send <- ErrResponse{Message: "Game already fulled"}
					break
				}
				if g.white == nil {
					g.white = conn
				} else {
					g.black = conn
				}
				h.gameConnections[conn] = join.GameID

				log.Println(conn, "join Game:",join.GameID,"Success!")
				res := newResponse(g.game, join.GameID, "Player join game")
				g.white.send <- res
				g.black.send <- res
			}


		case conn := <-h.unregister:
			log.Println(conn,"unregister")

		case m := <-h.move:
			conn,move := m.Conn, m.move
			log.Println(conn, "Plays", move)
			gameID, ok := h.gameConnections[conn]
			if !ok {
				conn.send <- ErrResponse{
					Message: "Not in a game",
				}
				break
			}
			g, ok := h.games[gameID]
			if !ok {
				conn.send <- ErrResponse{
					Message: "Invalid Game ID",
				}
				break
			}
			if g.white != conn && g.black != conn {
				conn.send <- ErrResponse{
					Message: "You're not a player in this game!",
				}
				break
			}
			if g.game.Position().Turn() == chess.White && g.white != conn {
				conn.send <- ErrResponse{
					Message: "Not your turn",
				}
				break
			}
			if err := g.game.MoveStr(move); err != nil {
				conn.send <- ErrResponse{
					Message: "Invalid Move",
				}
				break
			}
			//TODO: then check if move leads to something. e.g. checkmate
			res := newResponse(g.game, g.id, fmt.Sprintf("%s is played",move))
			g.white.send <- res
			g.black.send <- res
		}
	}
}

func newResponse(game *chess.Game,id string, event string) Response {
	return Response{
		GameID: id,
		Event: event, //TODO: event enum?
		FEN: game.FEN(),
		PGN: strings.TrimSpace(game.String()),
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
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
	id      string
	game    *chess.Game
	white   *Conn
	black   *Conn
	ongoing bool
}

type Games struct {
	games           map[string]*Game
	gameConnections map[*Conn]string
}

//whatever you want to send back
type Response struct {
	GameID string `json:"id"`
	Event  string `json:"event"`
	FEN    string `json:"fen"`
	PGN    string `json:"pgn"`
}

//Response in case of error
type ErrResponse struct {
	Message string `json:"message"`
}

type Move struct {
	Conn *Conn
	move string
}

type Join struct {
	Conn   *Conn
	GameID string
}

func (g *Games) create(conn *Conn) {
	log.Println("New Game Created By: ", conn)
	gameID := RandStringRunes(5)
	log.Println("Game ID:", gameID)
	g.games[gameID] = &Game{
		id:      gameID,
		game:    chess.NewGame(),
		white:   conn, //TODO: optionally create with color
		ongoing: true,
	}
	g.gameConnections[conn] = gameID
	conn.send <- newResponse(g.games[gameID].game, gameID, "Game Created")
}

func (g *Games) join(j Join) {
	conn, gameID := j.Conn, j.GameID
	log.Println(conn, "trying to join", gameID)
	if game, ok := g.games[gameID]; ok {
		if game.black != nil && game.white != nil {
			conn.send <- ErrResponse{Message: "Game already fulled"}
			return
		}
		var color string
		if game.white == nil {
			game.white = conn
			color = "white"
		} else {
			game.black = conn
			color = "black"
		}
		g.gameConnections[conn] = gameID

		log.Println(conn, "join Game:", gameID, "as", color)
		conn.send <- newResponse(game.game, gameID, "Game joined as "+color)
		res := newResponse(game.game, gameID, "Player join game")
		if color == "w" && game.black != nil {
			game.black.send <- res
		} else if color == "b" && game.white != nil {
			game.white.send <- res
		}

	}
}

func (g *Games) leave(conn *Conn) {
	log.Println(conn, "unregister")
	if id, ok := g.gameConnections[conn]; ok {
		if g, ok := g.games[id]; ok {
			if g.white == conn {
				g.white = nil
			} else {
				g.black = nil
			}
		}
		delete(g.gameConnections, conn)
	}
}

func (g *Games) move(m Move) {
	conn, move := m.Conn, m.move
	log.Println(conn, "Plays", move)
	gameID, ok := g.gameConnections[conn]
	if !ok {
		conn.send <- ErrResponse{
			Message: "Not in a game",
		}
		return
	}
	game, ok := g.games[gameID]
	if !ok {
		conn.send <- ErrResponse{
			Message: "Invalid Game ID",
		}
		return
	}
	if game.white != conn && game.black != conn {
		conn.send <- ErrResponse{
			Message: "You're not a player in this game!",
		}
		return
	}
	if game.game.Position().Turn() == chess.White && game.white != conn {
		conn.send <- ErrResponse{
			Message: "Not your turn",
		}
		return
	}
	if err := game.game.MoveStr(move); err != nil {
		conn.send <- ErrResponse{
			Message: "Invalid Move",
		}
		return
	}
	event := fmt.Sprintf("%s is played", move)
	if game.game.Method() != chess.NoMethod {
		event = fmt.Sprintf("Game Over: %v %v", game.game.Method(), game.game.Outcome())
		game.ongoing = false
		//TODO: delete the game or sth./ check g.ongoing b4 moving
	}
	res := newResponse(game.game, game.id, event)
	if game.white != nil {
		game.white.send <- res
	}
	if game.black != nil {
		game.black.send <- res
	}
}

func newResponse(game *chess.Game, id string, event string) *Response {
	return &Response{
		GameID: id,
		Event:  event, //TODO: event enum?
		FEN:    game.FEN(),
		PGN:    strings.TrimSpace(game.String()),
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

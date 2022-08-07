package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/notnil/chess"
)

type color string

const (
	white color = "white"
	black color = "black"
)

// for additional player fields
type Player struct {
	conn   *Conn
	gameID string
	color  color
	sdp    SDP
}

// allow multiple clients for a game, or kick out old ones?
type Game struct {
	id      string
	game    *chess.Game
	players map[color]Player
	ongoing bool
}

// store map of gameID to games
type GamesConnectionsMap struct {
	mu       sync.RWMutex
	gamesMap map[string]*Game
}

func (gm *GamesConnectionsMap) GetGame(gameID string) (Game, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	game, ok := gm.gamesMap[gameID]
	return *game, ok
}

func (gm *GamesConnectionsMap) CreateGame() string {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gameID := RandStringRunes(8)
	gm.gamesMap[gameID] = &Game{
		id:      gameID,
		game:    chess.NewGame(),
		players: make(map[color]Player),
		ongoing: true,
	}
	log.Printf("Creating a Game, gameID=%s\n", gameID)
	return gameID
}

func (gm *GamesConnectionsMap) JoinGame(gameID string, player Player) (Game, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	log.Printf("Joining a Game, gameID=%s color=%s\n", gameID, player.color)
	game, ok := gm.gamesMap[gameID]
	if !ok {
		return Game{}, fmt.Errorf("Game '%v' doesn't exists", gameID)
	}
	gm.gamesMap[gameID].players[player.color] = player
	return *game, nil
}

func (gm *GamesConnectionsMap) MakeMove(gameID string, move string) (Game, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	game := gm.gamesMap[gameID]
	log.Printf("Making a Move, gameID=%s move=%s\n", gameID, move)
	if err := game.game.MoveStr(move); err != nil {
		return Game{}, err
	}
	if game.game.Method() != chess.NoMethod {
		game.ongoing = false
	}
	//TODO: delete the game or sth./ check g.ongoing b4 moving
	return *game, nil
}

func (gm *GamesConnectionsMap) LeaveGame(player Player) {
	gm.mu.Lock()
	defer gm.mu.Lock()
	log.Printf("Leaving a Game, gameID=%s\n", player.gameID)
	if _, ok := gm.gamesMap[player.gameID]; ok {
		delete(gm.gamesMap[player.gameID].players, player.color)
	}
}

type UserGameConnectionsMap struct {
	mu               sync.RWMutex
	gamesConnections map[*Conn]Player //map connection to gameID, 1 game per connection
}

func (cm *UserGameConnectionsMap) Set(conn *Conn, player Player) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.gamesConnections[conn] = player
}

func (cm *UserGameConnectionsMap) UnSet(conn *Conn) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.gamesConnections, conn)
}

func (cm *UserGameConnectionsMap) Get(conn *Conn) (player Player, ok bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	player, ok = cm.gamesConnections[conn]
	return
}

//whatever you want to send back
type Response struct {
	GameID    string    `json:"id"`
	Event     string    `json:"event"`
	FEN       string    `json:"fen"`
	PGN       string    `json:"pgn"`
	RemoteSDP SDP       `json:"sdp,omitempty"`
	Candidate Candidate `json:"candidate,omitempty"`
}

//Response in case of error
type ErrResponse struct {
	Message string `json:"message"`
}

type CreateRequest struct {
	Conn *Conn
	SDP  SDP
}

type MoveRequest struct {
	Conn *Conn
	move string
}

type JoinRequest struct {
	Conn   *Conn
	GameID string
	SDP    SDP
}

type AnswerRequest struct {
	Conn *Conn
	SDP  SDP
}

type IceRequest struct {
	Conn      *Conn
	Candidate Candidate
}

// should connectionsMap be inside GamesConnectionsMap struct?
type GameService struct {
	gamesMap       GamesConnectionsMap
	connectionsMap UserGameConnectionsMap
}

func NewGameService() *GameService {
	return &GameService{
		gamesMap: GamesConnectionsMap{
			gamesMap: make(map[string]*Game),
		},
		connectionsMap: UserGameConnectionsMap{
			gamesConnections: make(map[*Conn]Player),
		},
	}
}

func (g *GameService) create(createReq CreateRequest) {
	conn := createReq.Conn
	gameID := g.gamesMap.CreateGame()
	player := Player{
		conn:   conn,
		gameID: gameID,
		color:  white, //TODO: optionally create with color
		sdp:    createReq.SDP,
	}
	g.gamesMap.JoinGame(gameID, player)
	g.connectionsMap.Set(conn, player)
	conn.send <- newResponse(chess.NewGame(), gameID, "Game Created")
}

func (g *GameService) join(j JoinRequest) {
	conn, gameID := j.Conn, j.GameID
	log.Println(conn, "trying to join", gameID)
	player := Player{
		conn:   conn,
		gameID: gameID,
		color:  black, //TODO: optionally create with color
		sdp:    j.SDP,
	}
	game, err := g.gamesMap.JoinGame(gameID, player)
	if err != nil {
		conn.send <- newErrorResponse(err.Error())
		return
	}

	g.connectionsMap.Set(conn, player)
	if white, ok := game.players[white]; ok {
		white.conn.send <- newResponse(game.game, gameID, "Player join game")
		conn.send <- newSDPResponse(game, fmt.Sprintf("Game joined as %s", black), white.sdp)
	}
}

func (g *GameService) leave(conn *Conn) {
	if player, ok := g.connectionsMap.Get(conn); ok {
		g.gamesMap.LeaveGame(player)
	}
}

func (g *GameService) move(m MoveRequest) {
	conn, move := m.Conn, m.move
	player, ok := g.connectionsMap.Get(conn)
	if !ok {
		conn.send <- newErrorResponse("Not in a game")
		return
	}

	// should the logic below be in  MakeMove func instead
	game, ok := g.gamesMap.GetGame(player.gameID)
	if !ok {
		conn.send <- newErrorResponse("Game does not exists")
		return
	}
	if !game.ongoing {
		conn.send <- newErrorResponse(fmt.Sprintf("Game Status: %v %v", game.game.Method(), game.game.Outcome()))
		return
	}

	inGame := false
	for _, p := range game.players {
		if p.conn == conn {
			inGame = true
			break
		}
	}
	if !inGame {
		conn.send <- newErrorResponse("You're not a player in this game!")
		return
	}

	if game.game.Position().Turn() == chess.White && player.color != white {
		conn.send <- newErrorResponse("Not your turn")
		return
	}

	game, err := g.gamesMap.MakeMove(player.gameID, move)
	if err != nil {
		conn.send <- newErrorResponse(err.Error())
		return
	}

	event := fmt.Sprintf("%s is played", move)
	if game.game.Method() != chess.NoMethod {
		event = fmt.Sprintf("Game Over: %v %v", game.game.Method(), game.game.Outcome())
	}
	res := newResponse(game.game, game.id, event)
	for _, player := range game.players {
		player.conn.send <- res
	}
}

func (g *GameService) answer(a AnswerRequest) {
	conn, sdp := a.Conn, a.SDP
	player, ok := g.connectionsMap.Get(conn)
	if !ok {
		conn.send <- newErrorResponse("Not in a game")
		return
	}

	game, ok := g.gamesMap.GetGame(player.gameID)
	if !ok {
		conn.send <- newErrorResponse("Game does not exists")
		return
	}

	toColor := white
	if player.color == white {
		toColor = black
	}

	if p, ok := game.players[toColor]; ok {
		p.conn.send <- newSDPResponse(game, "answer", sdp)
	}
}

func (g *GameService) ice(i IceRequest) {
	conn, candidate := i.Conn, i.Candidate
	player, ok := g.connectionsMap.Get(conn)
	if !ok {
		conn.send <- newErrorResponse("Not in a game")
		return
	}

	game, ok := g.gamesMap.GetGame(player.gameID)
	if !ok {
		conn.send <- newErrorResponse("Game does not exists")
		return
	}

	toColor := white
	if player.color == white {
		toColor = black
	}

	if p, ok := game.players[toColor]; ok {
		p.conn.send <- newICEResponse(game, candidate)
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

func newSDPResponse(game Game, event string, remoteSDP SDP) *Response {
	return &Response{
		GameID:    game.id,
		Event:     event,
		FEN:       game.game.FEN(),
		PGN:       strings.TrimSpace(game.game.String()),
		RemoteSDP: remoteSDP,
	}
}

func newICEResponse(game Game, candidate Candidate) *Response {
	return &Response{
		GameID:    game.id,
		Event:     "ice",
		FEN:       game.game.FEN(),
		PGN:       strings.TrimSpace(game.game.String()),
		Candidate: candidate,
	}
}
func newErrorResponse(msg string) ErrResponse {
	log.Printf("error: %s\n", msg)
	return ErrResponse{
		Message: msg,
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

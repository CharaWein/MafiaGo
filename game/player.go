package game

import (
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID       string
	Name     string
	Role     string
	Conn     *websocket.Conn
	Alive    bool
	VotedFor string
	IsReady  bool
}

func NewPlayer(name string, conn *websocket.Conn) *Player {
	return &Player{
		ID:      generateID(),
		Name:    name,
		Conn:    conn,
		Alive:   true,
		IsReady: false,
	}
}

func generateID() string {
	return "player_" + randString(8)
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

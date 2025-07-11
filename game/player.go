package game

import (
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID          string
	Name        string
	Role        string
	Conn        *websocket.Conn
	Alive       bool
	Ready       bool
	LastSeen    time.Time
	VotedFor    string
	NightAction string
}

func NewPlayer(name string, conn *websocket.Conn) *Player {
	return &Player{
		ID:       generateID(),
		Name:     name,
		Conn:     conn,
		Alive:    true,
		Ready:    false,
		LastSeen: time.Now(),
	}
}

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (p *Player) IsMafia() bool {
	return p.Role == RoleMafia || p.Role == RoleDon
}

func (p *Player) IsActive() bool {
	return time.Since(p.LastSeen) < 2*time.Minute
}

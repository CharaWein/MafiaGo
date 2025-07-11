package handlers

import (
	"sync"

	"github.com/CharaWein/mafia-game/game"
)

var (
	games      = make(map[string]*game.Game)
	gamesMutex sync.Mutex
)

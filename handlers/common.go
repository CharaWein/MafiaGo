package handlers

import (
	"sync"

	"github.com/CharaWein/mafia-game/game"
)

var (
	games      = make(map[string]*game.Game)
	gamesMutex sync.Mutex
)

// Handler структура для хранения состояния обработчиков
type Handler struct {
	// Можно добавить поля, если нужно хранить состояние
}

// NewHandler создает новый экземпляр Handler
func NewHandler() *Handler {
	return &Handler{}
}

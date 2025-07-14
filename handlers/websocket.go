package handlers

import (
	"log"
	"net/http"

	"github.com/CharaWein/mafia-game/game"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	gameID := r.URL.Query().Get("game_id")
	playerName := r.URL.Query().Get("name")

	gamesMutex.Lock()
	gameInstance, exists := games[gameID]
	gamesMutex.Unlock()

	if !exists {
		conn.WriteJSON(map[string]interface{}{"type": "error", "message": "Game not found"})
		return
	}

	player := game.NewPlayer(playerName, conn)
	gameInstance.AddPlayer(player)

	// Отправляем инициализационные данные
	conn.WriteJSON(map[string]interface{}{
		"type": "init",
		"payload": map[string]interface{}{
			"id":      player.ID,
			"players": gameInstance.GetPlayersList(),
		},
	})

	// Уведомляем других игроков
	gameInstance.BroadcastPlayersUpdate()

	for {
		var msg struct {
			Type    string      `json:"type"`
			Payload interface{} `json:"payload"`
		}

		if err := conn.ReadJSON(&msg); err != nil {
			gameInstance.RemovePlayer(player.ID)
			gameInstance.BroadcastPlayersUpdate()
			break
		}

		switch msg.Type {
		case "set_ready":
			if ready, ok := msg.Payload.(bool); ok {
				player.Ready = ready
				gameInstance.BroadcastPlayersUpdate()
			}
		}
	}
}

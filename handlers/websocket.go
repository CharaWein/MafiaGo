package handlers

import (
	"log"
	"net/http"
	"time"

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
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"payload": "Game not found",
		})
		return
	}

	player := game.NewPlayer(playerName, conn)
	gameInstance.AddPlayer(player)

	// Сразу отправляем текущее состояние лобби
	gameInstance.BroadcastLobbyState()

	// Уведомляем о статусе хоста
	isHost := len(gameInstance.Players) == 1
	if isHost {
		conn.WriteJSON(map[string]interface{}{
			"type": "host_status",
			"payload": map[string]bool{
				"isHost": true,
			},
		})
	}

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Read error: %v", err)
			gameInstance.RemovePlayer(player.ID)
			gameInstance.BroadcastLobbyState()
			break
		}

		player.LastSeen = time.Now()

		switch msg["type"] {
		case "set_ready":
			if ready, ok := msg["ready"].(bool); ok {
				player.Ready = ready
				gameInstance.BroadcastLobbyState()
			}
		case "start_game":
			if isHost {
				gameInstance.Start()
			}
		}
	}
}

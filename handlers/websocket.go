package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/CharaWein/mafia-game/game"
	"github.com/gorilla/websocket"
)

// Определяем upgrader на уровне пакета
var upgrader = websocket.Upgrader{
	CheckOrigin:      func(r *http.Request) bool { return true },
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	HandshakeTimeout: 10 * time.Second,
}

// Определяем структуру ChatMessage в пакете handlers
type ChatMessage struct {
	Sender string `json:"sender"`
	Text   string `json:"text"`
	Time   string `json:"time"`
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
			"isHost":  gameInstance.IsHost(player.ID),
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
				gameInstance.SetReadyStatus(player.ID, ready)
				gameInstance.BroadcastPlayersUpdate()
			}
		case "start_game":
			if gameInstance.IsHost(player.ID) && gameInstance.CanStartGame() {
				gameInstance.Start()
			}
		case "vote":
			if payload, ok := msg.Payload.(map[string]interface{}); ok {
				if target, ok := payload["target"].(string); ok {
					gameInstance.SetVote(player.ID, target)
				}
			}
		case "night_action":
			if payload, ok := msg.Payload.(map[string]interface{}); ok {
				if target, ok := payload["target"].(string); ok {
					gameInstance.SetNightAction(player.ID, target)
				}
			}
		case "chat":
			if payload, ok := msg.Payload.(map[string]interface{}); ok {
				if message, ok := payload["message"].(string); ok {
					gameInstance.Broadcast(game.Message{
						Type: "chat",
						Payload: ChatMessage{
							Sender: player.Name,
							Text:   message,
							Time:   time.Now().Format("15:04"),
						},
					})
				}
			}
		}
	}
}

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

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
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
		conn.WriteJSON(game.Message{Type: "error", Payload: "Game not found"})
		return
	}

	player := game.NewPlayer(playerName, conn)
	gameInstance.AddPlayer(player)

	for {
		var msg game.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Read error: %v", err)
			gameInstance.RemovePlayer(player.ID)
			break
		}

		player.LastSeen = time.Now()

		switch msg.Type {
		case "night_action":
			if target, ok := msg.Payload.(string); ok {
				gameInstance.SetNightAction(player.ID, target)
			}
		case "vote":
			if target, ok := msg.Payload.(string); ok {
				gameInstance.SetVote(player.ID, target)
			}
		case "chat":
			if text, ok := msg.Payload.(string); ok {
				broadcastChat(gameInstance, player, text)
			}
		case "ready":
			if ready, ok := msg.Payload.(bool); ok {
				gameInstance.SetReadyStatus(player.ID, ready)
			}
		}
	}
}

func broadcastChat(g *game.Game, sender *game.Player, text string) {
	msg := game.Message{
		Type: "chat",
		Payload: game.ChatMessage{
			Sender: sender.Name,
			Text:   text,
			Time:   time.Now().Format("15:04"),
		},
	}
	g.Broadcast(msg)
}

package handlers

import (
	"log"
	"net/http"

	"github.com/CharaWein/mafia-game/game"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	// Получаем ID игры из URL
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	gamesMutex.Lock()
	g, exists := games[gameID]
	gamesMutex.Unlock()

	if !exists {
		log.Printf("Game not found: %s", gameID)
		return
	}

	// Здесь должна быть логика подключения игрока через WebSocket
	// Например:
	// playerID := r.URL.Query().Get("player_id")
	// if player := g.Players[playerID]; player != nil {
	//     player.Conn = ws
	// }

	// Обработка сообщений
	for {
		var msg game.Message
		if err := ws.ReadJSON(&msg); err != nil {
			log.Printf("read error: %v", err)
			break
		}

		// Обработка сообщения
		handleWebSocketMessage(g, ws, msg)
	}
}

func handleWebSocketMessage(g *game.Game, ws *websocket.Conn, msg game.Message) {
	// Реализация обработки сообщений
}

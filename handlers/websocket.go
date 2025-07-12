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

func (h *Handler) getPlayerByConn(conn *websocket.Conn) *game.Player {
	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	for _, g := range games {
		for _, p := range g.Players {
			if p.Conn == conn {
				return p
			}
		}
	}
	return nil
}

func (h *Handler) getGameByPlayer(playerID string) *game.Game {
	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	for _, g := range games {
		if _, exists := g.Players[playerID]; exists {
			return g
		}
	}
	return nil
}

func (h *Handler) isHost(player *game.Player, game *game.Game) bool {
	if len(game.Players) == 0 {
		return false
	}
	for _, p := range game.Players {
		return p.ID == player.ID // Первый игрок - хост
	}
	return false
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
		conn.WriteJSON(game.Message{Type: "error", Payload: "Game not found"})
		return
	}

	player := game.NewPlayer(playerName, conn)
	gameInstance.AddPlayer(player)

	isHost := len(gameInstance.Players) == 1
	conn.WriteJSON(game.Message{
		Type: "host_status",
		Payload: map[string]bool{
			"is_host": isHost,
		},
	})

	gameInstance.Broadcast(game.Message{
		Type: "player_joined",
		Payload: game.PlayerInfo{
			ID:   player.ID,
			Name: player.Name,
		},
	})

	for {
		var msg game.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Игрок %s вышел: %v", player.Name, err)
			gameInstance.RemovePlayer(player.ID)
			gameInstance.Broadcast(game.Message{
				Type:    "player_left",
				Payload: player.ID,
			})
			break
		}

		player.LastSeen = time.Now()

		switch msg.Type {
		case "kick_player":
			if playerID, ok := msg.Payload.(string); ok && h.isHost(player, gameInstance) {
				gameInstance.RemovePlayer(playerID)
				if p, exists := gameInstance.Players[playerID]; exists && p.Conn != nil {
					p.Conn.WriteJSON(game.Message{
						Type:    "kicked",
						Payload: "Вас исключили из игры",
					})
					p.Conn.Close()
				}
				gameInstance.Broadcast(game.Message{
					Type:    "player_left",
					Payload: playerID,
				})
			}
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
		case "start_game":
			h.handleStartGame(conn)
		}
	}
}

func (h *Handler) handleStartGame(ws *websocket.Conn) {
	player := h.getPlayerByConn(ws)
	if player == nil {
		return
	}

	game := h.getGameByPlayer(player.ID)
	if game == nil {
		return
	}

	// Проверяем, что это ведущий (первый подключившийся игрок)
	if len(game.Players) > 0 {
		firstPlayerID := ""
		for id := range game.Players {
			firstPlayerID = id
			break
		}
		if firstPlayerID == player.ID {
			game.Start()
			h.broadcastGameState(game)
		}
	}
}

func (h *Handler) broadcastGameState(g *game.Game) {
	state := g.GetGameState()
	msg := game.Message{
		Type:    "game_state",
		Payload: state,
	}
	g.Broadcast(msg)
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

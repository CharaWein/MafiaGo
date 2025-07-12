func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	gameID := r.URL.Query().Get("game_id")
	playerName := r.URL.Query().Get("name")

	gamesMutex.Lock()
	game, exists := games[gameID]
	gamesMutex.Unlock()

	if !exists {
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"message": "Игра не найдена",
		})
		return
	}

	player := game.NewPlayer(playerName, conn)
	game.AddPlayer(player)

	// Отправляем текущий список игроков
	game.BroadcastPlayersList()

	// Уведомляем нового игрока, является ли он хостом
	isHost := len(game.Players) == 1
	conn.WriteJSON(map[string]interface{}{
		"type":   "host_status",
		"isHost": isHost,
	})

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			game.RemovePlayer(player.ID)
			game.BroadcastPlayersList()
			break
		}

		switch msg["type"] {
		case "set_ready":
			if ready, ok := msg["ready"].(bool); ok {
				player.Ready = ready
				game.BroadcastPlayersList()
			}
		case "start_game":
			if isHost {
				game.Start()
			}
		}
	}
}

func (g *Game) BroadcastPlayersList() {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	players := make([]map[string]interface{}, 0)
	for _, p := range g.Players {
		players = append(players, map[string]interface{}{
			"id":    p.ID,
			"name":  p.Name,
			"ready": p.Ready,
		})
	}

	msg := map[string]interface{}{
		"type":    "players_update",
		"players": players,
	}

	for _, p := range g.Players {
		if p.Conn != nil {
			p.Conn.WriteJSON(msg)
		}
	}
}
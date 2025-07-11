package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/CharaWein/mafia-game/game"
	"github.com/gorilla/mux"
)

func CreateGame(w http.ResponseWriter, r *http.Request) {
	g := game.NewGame()

	gamesMutex.Lock()
	games[g.ID] = g
	gamesMutex.Unlock()

	json.NewEncoder(w).Encode(map[string]string{
		"game_id": g.ID,
	})
}

func JoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	gamesMutex.Lock()
	g, exists := games[gameID]
	gamesMutex.Unlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	var data struct {
		PlayerName string `json:"player_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Добавляем игрока
	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	player := game.NewPlayer(data.PlayerName, nil) // nil - временно, потом заменим на WebSocket
	g.Players[player.ID] = player

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "joined",
		"game_id": gameID,
	})
}

func GetGameState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	gamesMutex.Lock()
	g, exists := games[gameID]
	gamesMutex.Unlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"game_id": gameID,
		"phase":   g.Phase,
		"players": len(g.Players),
		"day":     g.DayNumber,
	})
}

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

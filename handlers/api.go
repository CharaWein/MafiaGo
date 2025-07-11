package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/CharaWein/mafia-game/game"
	"github.com/gorilla/mux"
)

func CreateGame(w http.ResponseWriter, r *http.Request) {
	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	newGame := game.NewGame()
	games[newGame.ID] = newGame

	json.NewEncoder(w).Encode(map[string]string{
		"game_id": newGame.ID,
	})
}

func JoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	gamesMutex.Lock()
	_, exists := games[gameID]
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

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "joined",
		"game_id": gameID,
	})
}

func GetGameState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	gamesMutex.Lock()
	game, exists := games[gameID]
	gamesMutex.Unlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	state := game.GetGameState()
	json.NewEncoder(w).Encode(state)
}

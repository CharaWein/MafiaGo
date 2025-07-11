package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/CharaWein/mafia-game/game"
	"github.com/gorilla/mux"
)

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	g := game.NewGame()

	gamesMutex.Lock()
	games[g.ID] = g
	gamesMutex.Unlock()

	json.NewEncoder(w).Encode(map[string]string{
		"game_id": g.ID,
	})
}

func (h *Handler) JoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	var request struct {
		PlayerName string `json:"player_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	game, exists := games[gameID]
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	for _, p := range game.Players {
		if p.Name == request.PlayerName {
			http.Error(w, "Player with this name already exists", http.StatusConflict)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "joined",
		"game_id": gameID,
	})
}

func (h *Handler) GetGameState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameID"]

	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	game, exists := games[gameID]
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	state := game.GetGameState()
	json.NewEncoder(w).Encode(state)
}

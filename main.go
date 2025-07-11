package main

import (
	"log"
	"net/http"

	"github.com/CharaWein/mafia-game/handlers"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// API routes
	r.HandleFunc("/create", handlers.CreateGame).Methods("POST")
	r.HandleFunc("/join/{gameID}", handlers.JoinGame).Methods("POST")
	r.HandleFunc("/state/{gameID}", handlers.GetGameState).Methods("GET")

	// WebSocket
	r.HandleFunc("/ws", handlers.WebSocketHandler)

	// Static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

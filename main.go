package main

import (
	"log"
	"net/http"

	"github.com/CharaWein/mafia-game/handlers"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	h := handlers.NewHandler() // Используем короткое имя h вместо handler

	r.HandleFunc("/create", h.CreateGame).Methods("POST")
	r.HandleFunc("/join/{gameID}", h.JoinGame).Methods("POST")
	r.HandleFunc("/state/{gameID}", h.GetGameState).Methods("GET")
	r.HandleFunc("/ws", h.WebSocketHandler)

	// Static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

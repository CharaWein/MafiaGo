package main

import (
	"log"
	"net/http"

	"github.com/CharaWein/mafia-game/handlers"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	h := handlers.NewHandler()

	// Настройка маршрутов
	r.HandleFunc("/create", h.CreateGame).Methods("POST")
	r.HandleFunc("/join/{gameID}", h.JoinGame).Methods("POST")
	r.HandleFunc("/lobby/{gameID}", h.GetLobbyState).Methods("GET")
	r.HandleFunc("/ws", h.WebSocketHandler)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	port := ":8080"
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(port, r))
}

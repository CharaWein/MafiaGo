package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	r.HandleFunc("/ws/{gameID}", handlers.WebSocketHandler)

	// Serve static files from /static directory
	workDir, _ := os.Getwd()
	staticDir := filepath.Join(workDir, "static")
	fs := http.FileServer(http.Dir(staticDir))

	// Handle all requests with the file server
	r.PathPrefix("/").Handler(fs)

	// Special handler for favicon
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticDir, "favicon.ico"))
	})

	log.Println("Server starting on :8080")
	log.Println("Static files directory:", staticDir)
	log.Fatal(http.ListenAndServe(":8080", r))
}

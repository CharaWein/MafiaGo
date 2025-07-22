package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Player struct {
	Nickname string
	Ready    bool
	Conn     *websocket.Conn
}

type Room struct {
	ID      string
	Players map[string]*Player
	mu      sync.Mutex
}

var rooms = make(map[string]*Room)
var roomsMu sync.Mutex
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var templates *template.Template

func main() {
	// Инициализация шаблонов
	initTemplates()

	// Роуты
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/create", handleCreateRoom)
	http.HandleFunc("/game", handleGame)
	http.HandleFunc("/ws", handleWebSocket)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initTemplates() {
	templates = template.Must(template.ParseFiles(
		filepath.Join("templates", "index.html"),
		filepath.Join("templates", "game.html"),
	))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index.html", nil)
}

func handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	roomID := uuid.New().String()
	roomsMu.Lock()
	rooms[roomID] = &Room{
		ID:      roomID,
		Players: make(map[string]*Player),
	}
	roomsMu.Unlock()
	json.NewEncoder(w).Encode(map[string]string{"roomID": roomID})
}

func handleGame(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"RoomID":   r.URL.Query().Get("roomID"),
		"Nickname": r.URL.Query().Get("nickname"),
	}
	renderTemplate(w, "game.html", data)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket error:", err)
		return
	}
	defer conn.Close()

	roomID := r.URL.Query().Get("roomID")
	nickname := r.URL.Query().Get("nickname")

	roomsMu.Lock()
	room, exists := rooms[roomID]
	roomsMu.Unlock()

	if !exists {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Room not found"}`))
		return
	}

	// Добавление игрока
	room.mu.Lock()
	room.Players[nickname] = &Player{
		Nickname: nickname,
		Ready:    false,
		Conn:     conn,
	}
	room.mu.Unlock()

	broadcastPlayers(room)

	// Обработка сообщений
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			removePlayer(room, nickname)
			broadcastPlayers(room)
			break
		}

		var data map[string]interface{}
		if err := json.Unmarshal(msg, &data); err == nil {
			if ready, ok := data["ready"].(bool); ok {
				room.mu.Lock()
				room.Players[nickname].Ready = ready
				room.mu.Unlock()
				broadcastPlayers(room)
			}
		}
	}
}

func broadcastPlayers(room *Room) {
	room.mu.Lock()
	defer room.mu.Unlock()

	players := make([]map[string]interface{}, 0)
	for _, p := range room.Players {
		players = append(players, map[string]interface{}{
			"nickname": p.Nickname,
			"ready":    p.Ready,
		})
	}

	msg, _ := json.Marshal(map[string]interface{}{
		"type":    "players_update",
		"players": players,
	})

	for _, p := range room.Players {
		p.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func removePlayer(room *Room, nickname string) {
	room.mu.Lock()
	defer room.mu.Unlock()
	delete(room.Players, nickname)
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

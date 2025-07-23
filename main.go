package main

import (
	"encoding/json"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var rng *rand.Rand

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type Player struct {
	Nickname string
	Ready    bool
	Conn     *websocket.Conn
	Role     string
	Alive    bool
}

type Room struct {
	ID          string
	Players     map[string]*Player
	Creator     string
	GameStarted bool
	GamePhase   string
	DayNumber   int
	mu          sync.Mutex
}

var rooms = make(map[string]*Room)
var roomsMu sync.Mutex
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var templates *template.Template

func main() {
	initTemplates()

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
	nickname := r.URL.Query().Get("nickname")
	roomID := uuid.New().String()

	roomsMu.Lock()
	rooms[roomID] = &Room{
		ID:      roomID,
		Players: make(map[string]*Player),
		Creator: nickname,
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

	room.mu.Lock()
	if _, exists := room.Players[nickname]; exists {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Nickname already in use"}`))
		room.mu.Unlock()
		return
	}

	room.Players[nickname] = &Player{
		Nickname: nickname,
		Ready:    false,
		Conn:     conn,
		Alive:    true,
	}
	room.mu.Unlock()

	broadcastPlayers(room)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			removePlayer(room, nickname)
			broadcastPlayers(room)
			break
		}

		var data map[string]interface{}
		if err := json.Unmarshal(msg, &data); err == nil {
			switch data["type"] {
			case "player_ready":
				if ready, ok := data["ready"].(bool); ok {
					room.mu.Lock()
					room.Players[nickname].Ready = ready
					room.mu.Unlock()
					broadcastPlayers(room)
				}
			case "start_game":
				if nickname == room.Creator {
					startGame(room)
				}
			}
		}
	}
}

func broadcastPlayers(room *Room) {
	room.mu.Lock()
	defer room.mu.Unlock()

	players := make([]map[string]interface{}, 0)
	allReady := true
	readyCount := 0

	for _, p := range room.Players {
		players = append(players, map[string]interface{}{
			"nickname":  p.Nickname,
			"ready":     p.Ready,
			"isCreator": p.Nickname == room.Creator,
		})

		if p.Ready {
			readyCount++
		} else {
			allReady = false
		}
	}

	canStart := allReady && len(players) >= 4 && !room.GameStarted

	msg, _ := json.Marshal(map[string]interface{}{
		"type":        "players_update",
		"players":     players,
		"canStart":    canStart,
		"gameStarted": room.GameStarted,
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

func startGame(room *Room) {
	room.mu.Lock()
	if room.GameStarted {
		room.mu.Unlock()
		return
	}

	room.GameStarted = true
	players := make([]*Player, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, p)
	}
	room.mu.Unlock()

	assignRoles(players)
	notifyPlayers(room)
	startNightPhase(room)
}

func assignRoles(players []*Player) {
	count := len(players)
	if count < 4 {
		return
	}

	mafiaCount := (count - 5) / 2
	if mafiaCount < 1 {
		mafiaCount = 1
	}

	roles := make([]string, 0, count)
	roles = append(roles, "mafia_don")
	for i := 0; i < mafiaCount; i++ {
		roles = append(roles, "mafia")
	}
	roles = append(roles, "sheriff")
	for i := len(roles); i < count; i++ {
		roles = append(roles, "civilian")
	}

	rng.Shuffle(len(roles), func(i, j int) {
		roles[i], roles[j] = roles[j], roles[i]
	})

	for i, p := range players {
		p.Role = roles[i]
	}
}

func notifyPlayers(room *Room) {
	room.mu.Lock()
	defer room.mu.Unlock()

	for _, p := range room.Players {
		msg, _ := json.Marshal(map[string]interface{}{
			"type":         "game_started",
			"role":         p.Role,
			"playersCount": len(room.Players),
			"mafiaCount":   (len(room.Players)-5)/2 + 1, // +1 для дона
		})
		p.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func startNightPhase(room *Room) {
	room.mu.Lock()
	room.GamePhase = "night"
	room.DayNumber = 1
	room.mu.Unlock()

	for _, p := range room.Players {
		var msg []byte

		switch p.Role {
		case "mafia_don":
			msg, _ = json.Marshal(map[string]interface{}{
				"type":    "night_start",
				"phase":   "mafia",
				"message": "Вы Дон мафии. Выберите жертву для убийства",
				"isDon":   true,
			})
		case "mafia":
			msg, _ = json.Marshal(map[string]interface{}{
				"type":    "night_start",
				"phase":   "mafia",
				"message": "Вы мафия. Обсудите с Доном, кого убить",
				"isDon":   false,
			})
		case "sheriff":
			msg, _ = json.Marshal(map[string]interface{}{
				"type":    "night_start",
				"phase":   "sheriff",
				"message": "Вы Шериф. Выберите игрока для проверки",
			})
		default:
			msg, _ = json.Marshal(map[string]interface{}{
				"type":    "night_start",
				"phase":   "sleep",
				"message": "Ночь. Вы спите...",
			})
		}

		p.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

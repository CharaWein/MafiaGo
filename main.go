package main

import (
	"encoding/json"
	"html/template"
	"log"
	"math/rand/v2"
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
	Role     string // Добавляем поле Role
	Alive    bool
}

type Room struct {
	ID          string
	Players     map[string]*Player
	Creator     string
	GameStarted bool
	GamePhase   string // "night", "day", "voting"
	DayNumber   int
	mu          sync.Mutex
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

	// Добавляем игрока в комнату
	room.mu.Lock()
	if _, playerExists := room.Players[nickname]; playerExists {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Nickname already in use"}`))
		room.mu.Unlock()
		return
	}

	room.Players[nickname] = &Player{
		Nickname: nickname,
		Ready:    false,
		Conn:     conn,
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
				// ... существующий код ...
			case "start_game":
				if nickname == room.Creator {
					room.mu.Lock()
					room.GameStarted = true
					room.mu.Unlock()
					broadcastPlayers(room)
					startGame(room) // Новая функция для начала игры
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

	msg, _ := json.Marshal(map[string]interface{}{
		"type":        "players_update",
		"players":     players,
		"canStart":    allReady && readyCount >= 4 && !room.GameStarted,
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

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func startGame(room *Room) {
	// Подготовка списка игроков
	players := make([]*Player, 0, len(room.Players))
	for _, p := range room.Players {
		p.Alive = true // Все живы в начале игры
		players = append(players, p)
	}

	// Раздаем роли
	assignRoles(players)

	// Отправляем роли игрокам
	room.mu.Lock()
	defer room.mu.Unlock()

	for _, p := range room.Players {
		msg, _ := json.Marshal(map[string]interface{}{
			"type":         "game_started",
			"role":         p.Role,
			"playersCount": len(room.Players),
		})
		p.Conn.WriteMessage(websocket.TextMessage, msg)
	}

	// Начинаем первую ночь
	startNightPhase(room)
}

func assignRoles(players []*Player) {
	count := len(players)
	if count < 4 {
		return // Недостаточно игроков
	}

	// Количество мафий (без дона)
	mafiaCount := (count - 5) / 2
	if mafiaCount < 1 {
		mafiaCount = 1
	}

	// Дон мафии
	players[0].Role = "mafia_don"

	// Простые мафии
	for i := 1; i <= mafiaCount; i++ {
		players[i].Role = "mafia"
	}

	// Шериф
	players[mafiaCount+1].Role = "sheriff"

	// Остальные - мирные жители
	for i := mafiaCount + 2; i < count; i++ {
		players[i].Role = "civilian"
	}

	// Перемешиваем роли
	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})
}

func startNightPhase(room *Room) {
	// Отправляем сообщения в зависимости от роли
	for _, p := range room.Players {
		var msg []byte

		switch p.Role {
		case "mafia", "mafia_don":
			msg, _ = json.Marshal(map[string]interface{}{
				"type":    "night_start",
				"phase":   "mafia",
				"message": "Выберите жертву для убийства",
			})
		case "sheriff":
			msg, _ = json.Marshal(map[string]interface{}{
				"type":    "night_start",
				"phase":   "sheriff",
				"message": "Выберите игрока для проверки",
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

	// Можно добавить таймер для автоматического завершения фазы
}

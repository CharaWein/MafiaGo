package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // В продакшене замените на проверку origin!
	},
}

// Room представляет лобби/комнату
type Room struct {
	ID      string
	Players map[*websocket.Conn]bool
	mu      sync.Mutex
}

var rooms = make(map[string]*Room)
var roomsMu sync.Mutex

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/create", createRoom)
	http.HandleFunc("/list", listRooms)

	log.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Создание новой комнаты
func createRoom(w http.ResponseWriter, r *http.Request) {
	roomID := uuid.New().String()

	roomsMu.Lock()
	rooms[roomID] = &Room{
		ID:      roomID,
		Players: make(map[*websocket.Conn]bool),
	}
	roomsMu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"roomID": roomID})
}

// Список всех комнат
func listRooms(w http.ResponseWriter, r *http.Request) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	var roomIDs []string
	for id := range rooms {
		roomIDs = append(roomIDs, id)
	}

	json.NewEncoder(w).Encode(map[string][]string{"rooms": roomIDs})
}

// Обработчик WebSocket
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка WebSocket:", err)
		return
	}
	defer conn.Close()

	roomID := r.URL.Query().Get("roomID")
	if roomID == "" {
		log.Println("Не указан roomID")
		return
	}

	roomsMu.Lock()
	room, exists := rooms[roomID]
	roomsMu.Unlock()

	if !exists {
		log.Println("Комната не найдена")
		return
	}

	// Добавляем игрока в комнату
	room.mu.Lock()
	room.Players[conn] = true
	room.mu.Unlock()

	log.Printf("Игрок подключился к комнате %s", roomID)

	// Чтение сообщений от игрока (можно расширить)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Игрок отключился:", err)
			room.mu.Lock()
			delete(room.Players, conn)
			room.mu.Unlock()
			break
		}
		log.Printf("Сообщение от игрока в комнате %s: %s", roomID, string(msg))
	}
}

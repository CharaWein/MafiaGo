package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Для тестов разрешаем все origin
	},
}

type Room struct {
	ID      string
	Players map[*websocket.Conn]bool
	mu      sync.Mutex
}

var rooms = make(map[string]*Room)
var roomsMu sync.Mutex

func main() {
	// Главная страница с формой для ника и комнаты
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("index").Parse(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>MafiaGo</title>
				<style>
					body { font-family: Arial, sans-serif; margin: 20px; }
					input, button { padding: 8px; margin: 5px; }
				</style>
			</head>
			<body>
				<h1>Добро пожаловать в MafiaGo!</h1>
				<form id="joinForm">
					<input type="text" id="nickname" placeholder="Ваш ник" required>
					<input type="text" id="roomID" placeholder="ID комнаты (оставьте пустым для создания новой)">
					<button type="submit">Присоединиться</button>
				</form>
				<script>
					document.getElementById("joinForm").addEventListener("submit", async (e) => {
						e.preventDefault();
						const nickname = document.getElementById("nickname").value;
						const roomID = document.getElementById("roomID").value;
						
						if (!roomID) {
							// Создаём новую комнату
							const response = await fetch("/create");
							const data = await response.json();
							window.location.href = `/game.html?roomID=${data.roomID}&nickname=${nickname}`;
						} else {
							// Присоединяемся к существующей
							window.location.href = `/game.html?roomID=${roomID}&nickname=${nickname}`;
						}
					});
				</script>
			</body>
			</html>
		`))
		tmpl.Execute(w, nil)
	})

	// API для создания комнаты
	http.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		roomID := uuid.New().String()
		roomsMu.Lock()
		rooms[roomID] = &Room{
			ID:      roomID,
			Players: make(map[*websocket.Conn]bool),
		}
		roomsMu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"roomID": roomID})
	})

	// WebSocket-подключение
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket error:", err)
			return
		}
		defer conn.Close()

		roomID := r.URL.Query().Get("roomID")
		if roomID == "" {
			log.Println("RoomID not specified")
			return
		}

		roomsMu.Lock()
		room, exists := rooms[roomID]
		roomsMu.Unlock()

		if !exists {
			log.Println("Room not found")
			return
		}

		room.mu.Lock()
		room.Players[conn] = true
		room.mu.Unlock()

		log.Printf("New player in room %s", roomID)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Player disconnected:", err)
				room.mu.Lock()
				delete(room.Players, conn)
				room.mu.Unlock()
				break
			}
			log.Printf("Message from room %s: %s", roomID, string(msg))
		}
	})

	// Статические файлы (если нужно)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
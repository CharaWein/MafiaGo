package game

type MessageType string

const (
	MsgGameState    MessageType = "game_state"
	MsgChat         MessageType = "chat"
	MsgPlayerJoined MessageType = "player_joined"
	MsgNightStart   MessageType = "night_start"
	MsgDayStart     MessageType = "day_start"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type ChatMessage struct {
	Sender string `json:"sender"`
	Text   string `json:"text"`
	Time   string `json:"time"`
}

type GameState struct {
	Phase     string       `json:"phase"`
	DayNumber int          `json:"day_number"`
	Players   []PlayerInfo `json:"players"`
	Winner    string       `json:"winner,omitempty"`
}

type PlayerInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alive bool   `json:"alive"`
	Role  string `json:"role,omitempty"`
	Ready bool   `json:"ready"`
}

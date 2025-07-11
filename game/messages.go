package game

type MessageType string

const (
	MsgGameState    MessageType = "game_state"
	MsgPlayerJoined MessageType = "player_joined"
	MsgChatMessage  MessageType = "chat_message"
	MsgVote         MessageType = "vote"
	MsgNightAction  MessageType = "night_action"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
	Error   string      `json:"error,omitempty"`
}

type GameState struct {
	Phase     string         `json:"phase"`
	DayNumber int            `json:"day_number"`
	Players   []PlayerPublic `json:"players"`
	Winner    string         `json:"winner,omitempty"`
	YourRole  string         `json:"your_role,omitempty"`
}

type PlayerPublic struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alive bool   `json:"alive"`
	Role  string `json:"role,omitempty"` // Only visible if revealed
}

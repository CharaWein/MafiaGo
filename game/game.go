package game

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

type Game struct {
	ID           string
	Players      map[string]*Player
	Mu           sync.Mutex
	Phase        string // "lobby", "night", "day", "ended"
	DayNumber    int
	Winner       string // "mafia", "civilians"
	CreatedAt    time.Time
	Votes        map[string]int    // Для дневного голосования
	NightActions map[string]string // Для ночных действий
}

func generateGameID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func NewGame() *Game {
	return &Game{
		ID:           generateGameID(),
		Players:      make(map[string]*Player),
		Phase:        "lobby",
		DayNumber:    0,
		CreatedAt:    time.Now(),
		Votes:        make(map[string]int),
		NightActions: make(map[string]string),
	}
}

func (g *Game) Start() {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	g.assignRoles()
	g.Phase = "night"
	g.DayNumber = 1
	g.notifyAll()
}

// Реализация недостающих методов:

func (g *Game) getMafiaVote() string {
	votes := make(map[string]int)
	for _, action := range g.NightActions {
		if action != "" {
			votes[action]++
		}
	}

	maxVotes := 0
	var target string
	for playerID, count := range votes {
		if count > maxVotes {
			maxVotes = count
			target = playerID
		}
	}
	return target
}

func (g *Game) getSheriffCheck() string {
	for playerID, p := range g.Players {
		if p.Role == RoleSheriff {
			return g.NightActions[playerID]
		}
	}
	return ""
}

func (g *Game) getDonCheck() string {
	for playerID, p := range g.Players {
		if p.Role == RoleDon {
			return g.NightActions[playerID]
		}
	}
	return ""
}

func (g *Game) getMajorityVote() string {
	maxVotes := 0
	var target string

	for playerID, count := range g.Votes {
		if count > maxVotes {
			maxVotes = count
			target = playerID
		} else if count == maxVotes {
			// При ничьей никто не умирает
			target = ""
		}
	}

	if maxVotes == 0 {
		return ""
	}
	return target
}

func (g *Game) notifyAll() {
	for _, p := range g.Players {
		if p.Conn != nil {
			state := g.gameStateForPlayer(p)
			p.Conn.WriteJSON(state)
		}
	}
}

func (g *Game) gameStateForPlayer(p *Player) Message {
	publicPlayers := make([]PlayerPublic, 0)
	for _, player := range g.Players {
		publicPlayers = append(publicPlayers, PlayerPublic{
			ID:    player.ID,
			Name:  player.Name,
			Alive: player.Alive,
			Role:  g.getRevealedRole(p, player),
		})
	}

	return Message{
		Type: MsgGameState,
		Payload: GameState{
			Phase:     g.Phase,
			DayNumber: g.DayNumber,
			Players:   publicPlayers,
			Winner:    g.Winner,
			YourRole:  p.Role,
		},
	}
}

func (g *Game) getRevealedRole(currentPlayer, targetPlayer *Player) string {
	// Показываем роль только если:
	// 1. Игра закончена
	// 2. Игрок мертв
	// 3. Это шериф проверял мафию/дона
	if g.Phase == "ended" || !targetPlayer.Alive {
		return targetPlayer.Role
	}

	// Шериф видит результаты своих проверок
	if currentPlayer.Role == RoleSheriff {
		if check, exists := g.NightActions[currentPlayer.ID]; exists && check == targetPlayer.ID {
			if targetPlayer.Role == RoleMafia || targetPlayer.Role == RoleDon {
				return "mafia"
			}
			return "civilian"
		}
	}

	return ""
}

func (g *Game) assignRoles() {
	playerCount := len(g.Players)
	if playerCount < 4 {
		return // Недостаточно игроков
	}

	// Вычисляем количество мафии (без дона)
	mafiaCount := int(math.Floor(float64(playerCount-5) / 2))
	if mafiaCount < 1 {
		mafiaCount = 1
	}

	// Создаем список ID игроков
	playerIDs := make([]string, 0, len(g.Players))
	for id := range g.Players {
		playerIDs = append(playerIDs, id)
	}

	// Перемешиваем игроков
	rand.Shuffle(len(playerIDs), func(i, j int) {
		playerIDs[i], playerIDs[j] = playerIDs[j], playerIDs[i]
	})

	// Распределяем роли
	roles := make([]string, 0)
	roles = append(roles, RoleDon)
	for i := 0; i < mafiaCount; i++ {
		roles = append(roles, RoleMafia)
	}
	roles = append(roles, RoleSheriff)
	for i := len(roles); i < len(playerIDs); i++ {
		roles = append(roles, RoleCivilian)
	}

	// Назначаем роли
	for i, id := range playerIDs {
		if i < len(roles) {
			g.Players[id].Role = roles[i]
		} else {
			g.Players[id].Role = RoleCivilian
		}
	}
}

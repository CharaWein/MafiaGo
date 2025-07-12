package game

import (
	"math/rand"
	"sync"
	"time"
)

const (
	gameIDChars  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	gameIDLength = 6
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateGameID() string {
	b := make([]byte, gameIDLength)
	for i := range b {
		b[i] = gameIDChars[rand.Intn(len(gameIDChars))]
	}
	return string(b)
}

type Game struct {
	ID           string
	Players      map[string]*Player
	Mu           sync.Mutex
	Phase        string
	DayNumber    int
	Winner       string
	CreatedAt    time.Time
	Votes        map[string]string
	NightActions map[string]string
	readyPlayers int
}

type LobbyState struct {
	Players  []PlayerInfo `json:"players"`
	CanStart bool         `json:"canStart"`
}

func NewGame() *Game {
	return &Game{
		ID:           generateGameID(),
		Players:      make(map[string]*Player),
		Phase:        "lobby",
		DayNumber:    0,
		CreatedAt:    time.Now(),
		Votes:        make(map[string]string),
		NightActions: make(map[string]string),
		readyPlayers: 0,
	}
}

func (g *Game) AddPlayer(p *Player) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	g.Players[p.ID] = p
}

func (g *Game) RemovePlayer(playerID string) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	delete(g.Players, playerID)
}

func (g *Game) Broadcast(msg Message) error {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	var lastErr error
	for _, p := range g.Players {
		if p.Conn != nil {
			if err := p.Conn.WriteJSON(msg); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

func (g *Game) getAlivePlayers() []*Player {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	var alive []*Player
	for _, p := range g.Players {
		if p.Alive {
			alive = append(alive, p)
		}
	}
	return alive
}

func (g *Game) getAliveCount() int {
	return len(g.getAlivePlayers())
}

func (g *Game) getMafiaTarget() string {
	votes := make(map[string]int)
	for _, p := range g.Players {
		if p.IsMafia() && p.Alive && g.NightActions[p.ID] != "" {
			votes[g.NightActions[p.ID]]++
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

func (g *Game) SetNightAction(playerID, targetID string) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	g.NightActions[playerID] = targetID
}

func (g *Game) SetVote(playerID, targetID string) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	g.Votes[playerID] = targetID
}

func (g *Game) SetReadyStatus(playerID string, ready bool) {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	if p, exists := g.Players[playerID]; exists {
		if p.Ready != ready {
			p.Ready = ready
			if ready {
				g.readyPlayers++
			} else {
				g.readyPlayers--
			}
		}
	}
}

func (g *Game) getSheriffID() string {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	for id, p := range g.Players {
		if p.Role == RoleSheriff && p.Alive {
			return id
		}
	}
	return ""
}

func (g *Game) getDonID() string {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	for id, p := range g.Players {
		if p.Role == RoleDon && p.Alive {
			return id
		}
	}
	return ""
}

func (g *Game) checkGameEnd() bool {
	aliveMafia := 0
	aliveCivilians := 0

	for _, p := range g.getAlivePlayers() {
		if p.IsMafia() {
			aliveMafia++
		} else {
			aliveCivilians++
		}
	}

	if aliveMafia == 0 {
		g.Winner = "civilians"
		return true
	}

	if aliveMafia >= aliveCivilians {
		g.Winner = "mafia"
		return true
	}

	return false
}

func (g *Game) notifyAll() {
	state := g.GetGameState()
	msg := Message{
		Type:    MsgGameState,
		Payload: state,
	}
	g.Broadcast(msg)
}

func (g *Game) GetGameState() GameState {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	var players []PlayerInfo
	for _, p := range g.Players {
		players = append(players, PlayerInfo{
			ID:    p.ID,
			Name:  p.Name,
			Alive: p.Alive,
			Role:  p.Role,
			Ready: p.Ready,
		})
	}

	return GameState{
		Phase:     g.Phase,
		DayNumber: g.DayNumber,
		Players:   players,
		Winner:    g.Winner,
	}
}

func (g *Game) assignRoles() {
	// Реализация распределения ролей
	playerCount := len(g.Players)
	if playerCount < 4 {
		return // Минимум 4 игрока для игры
	}

	// Создаем список ID игроков
	playerIDs := make([]string, 0, playerCount)
	for id := range g.Players {
		playerIDs = append(playerIDs, id)
	}

	// Перемешиваем ID
	rand.Shuffle(playerCount, func(i, j int) {
		playerIDs[i], playerIDs[j] = playerIDs[j], playerIDs[i]
	})

	// Распределяем роли
	for i, id := range playerIDs {
		switch i {
		case 0:
			g.Players[id].Role = RoleDon
		case 1:
			g.Players[id].Role = RoleSheriff
		default:
			if i < playerCount/3+1 {
				g.Players[id].Role = RoleMafia
			} else {
				g.Players[id].Role = RoleCivilian
			}
		}
	}
}

func (g *Game) Start() {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	if g.Phase != "lobby" {
		return
	}

	g.assignRoles()
	g.Phase = "night"
	g.DayNumber = 1

	// Отправляем роли всем игрокам
	for _, p := range g.Players {
		if p.Conn != nil {
			p.Conn.WriteJSON(Message{
				Type: "role_assigned",
				Payload: map[string]interface{}{
					"role": p.Role,
				},
			})
		}
	}

	// Запускаем ночную фазу
	g.startNightPhase()
}

func (g *Game) startNightPhase() {
	// Отправляем уведомление о начале ночи
	msg := Message{
		Type: "night_start",
		Payload: map[string]interface{}{
			"number": g.DayNumber,
		},
	}
	g.Broadcast(msg)

	// Через 1 минуту автоматически завершаем ночь
	time.AfterFunc(1*time.Minute, func() {
		g.endNightPhase()
	})
}

func (g *Game) endNightPhase() {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	// Обрабатываем ночные действия
	target := g.getMafiaTarget()
	if target != "" {
		g.Players[target].Alive = false
	}

	// Переходим к дневной фазе
	g.Phase = "day"
	g.NightActions = make(map[string]string)

	// Отправляем уведомление о начале дня
	msg := Message{
		Type: "day_start",
		Payload: map[string]interface{}{
			"number": g.DayNumber,
			"killed": target,
		},
	}
	g.Broadcast(msg)

	// Через 2 минуты автоматически завершаем день
	time.AfterFunc(2*time.Minute, func() {
		g.endDayPhase()
	})
}

func (g *Game) endDayPhase() {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	// Обрабатываем дневные голоса
	g.processDayVotes()

	// Проверяем условие окончания игры
	if g.checkGameEnd() {
		msg := Message{
			Type: "game_end",
			Payload: map[string]interface{}{
				"winner": g.Winner,
			},
		}
		g.Broadcast(msg)
		return
	}

	// Переходим к следующей ночи
	g.DayNumber++
	g.Phase = "night"
	g.Votes = make(map[string]string)

	// Отправляем уведомление о начале ночи
	msg := Message{
		Type: "night_start",
		Payload: map[string]interface{}{
			"number": g.DayNumber,
		},
	}
	g.Broadcast(msg)

	// Через 1 минуту автоматически завершаем ночь
	time.AfterFunc(1*time.Minute, func() {
		g.endNightPhase()
	})
}

func (g *Game) processDayVotes() {
	voteCount := make(map[string]int)
	for _, target := range g.Votes {
		if target != "" {
			voteCount[target]++
		}
	}

	maxVotes := 0
	var toKill string
	for target, count := range voteCount {
		if count > maxVotes {
			maxVotes = count
			toKill = target
		} else if count == maxVotes {
			toKill = "" // Ничья - никто не умирает
		}
	}

	// Игрок погибает, если за него проголосовало больше половины живых игроков
	if toKill != "" && maxVotes > g.getAliveCount()/2 {
		g.Players[toKill].Alive = false
		g.Broadcast(Message{
			Type: "player_killed",
			Payload: map[string]interface{}{
				"player_id": toKill,
				"day":       g.DayNumber,
			},
		})
	}
}

func (g *Game) GetLobbyState() LobbyState {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	var players []PlayerInfo
	for _, p := range g.Players {
		players = append(players, PlayerInfo{
			ID:    p.ID,
			Name:  p.Name,
			Ready: p.Ready,
		})
	}

	return LobbyState{
		Players:  players,
		CanStart: len(players) >= 4 && g.readyPlayers == len(players),
	}
}

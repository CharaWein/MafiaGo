class MafiaGame {
    constructor() {
        this.socket = null;
        this.gameId = null;
        this.playerId = null;
        this.playerName = null;
        this.isHost = false;
        this.role = null;
        this.lobbyUpdateInterval = null;
        
        this.initEventListeners();

        this.players = []; // Храним список игроков
    }
    
    initEventListeners() {
        document.getElementById('create-game-btn').addEventListener('click', () => this.createGame());
        document.getElementById('join-game-btn').addEventListener('click', () => this.joinGame());
        document.getElementById('send-message-btn').addEventListener('click', () => this.sendChatMessage());
        document.getElementById('ready-btn').addEventListener('click', () => this.toggleReady());
        
        const startBtn = document.getElementById('start-game-btn');
        if (startBtn) {
            startBtn.addEventListener('click', () => this.startGame());
        }
    }

    async createGame() {
        try {
            const response = await fetch('/create', { method: 'POST' });
            const data = await response.json();
            
            this.gameId = data.game_id;
            document.getElementById('game-id-display').textContent = this.gameId;
            document.getElementById('game-id-input').value = this.gameId;
            this.showMessage(`Игра создана! ID: ${this.gameId}`);
        } catch (error) {
            this.showMessage(`Ошибка при создании игры: ${error}`);
        }
    }
    
    async joinGame() {
        this.gameId = document.getElementById('game-id-input').value.trim();
        this.playerName = document.getElementById('player-name-input').value.trim();
        
        if (!this.gameId || !this.playerName) {
            this.showMessage('Введите ID игры и ваше имя');
            return;
        }
        
        try {
            const response = await fetch(`/join/${this.gameId}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ player_name: this.playerName })
            });
            
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText || 'Ошибка при присоединении к игре');
            }
            
            this.connectWebSocket();
            document.getElementById('lobby-screen').classList.add('hidden');
            document.getElementById('game-screen').classList.remove('hidden');
            this.showMessage(`Вы присоединились к игре как ${this.playerName}`);
            
            this.startLobbyUpdates();
        } catch (error) {
            this.showMessage(`Ошибка: ${error.message}`);
            console.error('Join game error:', error);
        }
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        const wsUrl = `${protocol}${window.location.host}/ws?game_id=${this.gameId}&name=${encodeURIComponent(this.playerName)}`;
        
        this.socket = new WebSocket(wsUrl);

        this.socket.onopen = () => {
            this.socket.send(JSON.stringify({
                type: 'get_players'
            }));
        };

        this.socket.onclose = () => {
            this.showMessage('Соединение с сервером потеряно');
        };

        this.socket.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            
            switch(msg.type) {
                case 'host_status':
                    this.isHost = msg.payload.is_host;
                    if (this.isHost) {
                        document.getElementById('start-game-btn').classList.remove('hidden');
                    }
                    break;
                case 'game_state':
                    this.updateGameState(msg.payload);
                    break;
                case 'role_assigned':
                    this.handleRoleAssignment(msg.payload.role);
                    break;
                case 'night_start':
                    this.showNightInterface();
                    break;
                case 'day_start':
                    this.showDayInterface(msg.payload.killed);
                    break;
                case 'player_killed':
                    this.showKilledPlayer(msg.payload);
                    break;
                case 'game_end':
                    this.showGameEnd(msg.payload.winner);
                    break;
                case 'chat_message':
                    this.addChatMessage(msg.payload);
                    break;
                case 'player_joined':
                    this.handlePlayerJoined(msg.payload);
                    break;
                case 'player_left':
                    this.handlePlayerLeft(msg.payload);
                    break;
                case 'kicked':
                    alert(msg.payload);
                    location.reload();
                    break;
                default:
                    console.log('Неизвестный тип сообщения:', msg);
            }
        };
    }

    updateGameState(state) {
        document.getElementById('game-phase').textContent = state.phase === 'night' ? 'Ночь' : 'День';
        document.getElementById('game-day').textContent = state.day_number;
        
        const playersList = document.getElementById('players-list');
        playersList.innerHTML = '';
        
        state.players.forEach(player => {
            const playerCard = document.createElement('div');
            playerCard.className = 'player-card';
            playerCard.innerHTML = `
                <div class="player-name">${player.name}</div>
                <div class="player-status">${player.alive ? 'Жив' : 'Мертв'}</div>
                ${player.role && !player.alive ? `<div class="player-role">${this.getRoleName(player.role)}</div>` : ''}
            `;
            
            if (player.alive) {
                playerCard.dataset.playerId = player.id;
                playerCard.addEventListener('click', () => this.handlePlayerSelection(player.id));
            }
            
            playersList.appendChild(playerCard);
        });
    }

    handleRoleAssignment(role) {
        this.role = role;
        const roleInfo = document.getElementById('role-info');
        roleInfo.querySelector('#role-name').textContent = this.getRoleName(role);
        roleInfo.classList.remove('hidden');
        
        // Скрываем все панели действий
        document.getElementById('mafia-panel').classList.add('hidden');
        document.getElementById('sheriff-panel').classList.add('hidden');
        
        // Показываем соответствующую панель действий
        if (role === 'don' || role === 'mafia') {
            document.getElementById('mafia-panel').classList.remove('hidden');
        } else if (role === 'sheriff') {
            document.getElementById('sheriff-panel').classList.remove('hidden');
        }
    }

    getRoleName(role) {
        const roleNames = {
            'don': 'Дон мафии',
            'mafia': 'Мафия',
            'sheriff': 'Шериф',
            'civilian': 'Мирный житель'
        };
        return roleNames[role] || role;
    }

    showNightInterface() {
        document.getElementById('game-phase').textContent = 'Ночь';
        document.getElementById('action-panel').classList.remove('hidden');
        
        if (this.role === 'mafia' || this.role === 'don') {
            this.showMessage('Выберите игрока для убийства');
        } else if (this.role === 'sheriff') {
            this.showMessage('Выберите игрока для проверки');
        } else {
            this.showMessage('Ночь - спите!');
        }
    }

    showDayInterface(killedPlayerId) {
        document.getElementById('game-phase').textContent = 'День';
        if (killedPlayerId) {
            this.showMessage(`Ночью был убит игрок ${this.getPlayerName(killedPlayerId)}`);
        }
        this.showMessage('Обсудите и проголосуйте за казнь подозреваемого');
    }

    showGameEnd(winner) {
        const winnerText = winner === 'mafia' ? 'Мафия победила!' : 'Мирные жители победили!';
        document.getElementById('winner-message').textContent = winnerText;
        document.getElementById('game-end').classList.remove('hidden');
    }

    handlePlayerSelection(playerId) {
        if (!this.socket) return;

        const currentPhase = document.getElementById('game-phase').textContent;
        
        if (currentPhase === 'Ночь') {
            if (this.role === 'mafia' || this.role === 'don') {
                this.socket.send(JSON.stringify({
                    type: 'night_action',
                    target: playerId
                }));
                this.showMessage(`Выбрана цель: ${this.getPlayerName(playerId)}`);
            } else if (this.role === 'sheriff') {
                this.socket.send(JSON.stringify({
                    type: 'sheriff_check',
                    target: playerId
                }));
                this.showMessage(`Проверяем игрока: ${this.getPlayerName(playerId)}`);
            }
        } else if (currentPhase === 'День') {
            this.socket.send(JSON.stringify({
                type: 'vote',
                target: playerId
            }));
            this.showMessage(`Вы проголосовали против ${this.getPlayerName(playerId)}`);
        }
    }

    getPlayerName(playerId) {
        // В реальной реализации нужно получать имя из состояния игры
        return playerId; // временная заглушка
    }

    sendChatMessage() {
        const input = document.getElementById('chat-input');
        const message = input.value.trim();
        
        if (message && this.socket) {
            this.socket.send(JSON.stringify({
                type: 'chat_message',
                text: message
            }));
            input.value = '';
        }
    }

    addChatMessage(message) {
        const chat = document.getElementById('chat-messages');
        const messageElement = document.createElement('div');
        messageElement.className = 'chat-message';
        messageElement.innerHTML = `
            <strong>${message.sender}:</strong> ${message.text}
            <span class="chat-time">${message.time || ''}</span>
        `;
        chat.appendChild(messageElement);
        chat.scrollTop = chat.scrollHeight;
    }

    toggleReady() {
        if (this.socket) {
            const readyBtn = document.getElementById('ready-btn');
            const isReady = readyBtn.textContent.includes('Готов');
            this.setReadyStatus(!isReady);
            readyBtn.textContent = isReady ? 'Я готов' : 'Отменить готовность';
            readyBtn.classList.toggle('ready', !isReady);
        }
    }

    setReadyStatus(ready) {
        if (this.socket) {
            this.socket.send(JSON.stringify({
                type: 'ready',
                payload: ready
            }));
        }
    }

    startGame() {
        if (this.socket && this.isHost) {
            this.socket.send(JSON.stringify({
                type: 'start_game'
            }));
        }
    }

    startLobbyUpdates() {
        this.stopLobbyUpdates();
        this.lobbyUpdateInterval = setInterval(() => this.updateLobby(), 2000);
    }

    stopLobbyUpdates() {
        if (this.lobbyUpdateInterval) {
            clearInterval(this.lobbyUpdateInterval);
            this.lobbyUpdateInterval = null;
        }
    }

    async updateLobby() {
        try {
            const response = await fetch(`/lobby/${this.gameId}`);
            if (!response.ok) return;
            
            const lobby = await response.json();
            this.renderLobbyPlayers(lobby.players);
            
            if (this.isHost) {
                const startBtn = document.getElementById('start-game-btn');
                startBtn.disabled = !lobby.canStart;
            }
        } catch (error) {
            console.error('Lobby update error:', error);
        }
    }

    showMessage(text) {
        const messageBox = document.getElementById('game-message');
        if (messageBox) {
            messageBox.textContent = text;
            messageBox.classList.remove('hidden');
            setTimeout(() => messageBox.classList.add('hidden'), 3000);
        }
        console.log(text);
    }

    getPlayerList() {
        return this.players.map(player => ({
            id: player.id,
            name: player.name,
            ready: player.ready || false
        }));
    }

    updatePlayerList(players) {
        this.players = players;
        this.renderLobbyPlayers(players);
    }

    handlePlayerJoined(player) {
        this.players.push(player);
        this.renderLobbyPlayers(this.players);
    }

    handlePlayerLeft(playerId) {
        this.players = this.players.filter(p => p.id !== playerId);
        this.renderLobbyPlayers(this.players);
    }

    renderLobbyPlayers(players) {
        const container = document.getElementById('lobby-players');
        container.innerHTML = '';
        
        players.forEach(player => {
            const playerEl = document.createElement('div');
            playerEl.className = `player ${player.ready ? 'ready' : ''}`;
            playerEl.innerHTML = `
                <span class="player-name">${player.name}</span>
                <span class="player-status">${player.ready ? '✓ Готов' : 'Не готов'}</span>
                ${this.isHost && !player.ready ? `<button class="kick-btn" data-id="${player.id}">✖</button>` : ''}
            `;
            container.appendChild(playerEl);
        });

        // Добавляем обработчики для кнопок исключения
        document.querySelectorAll('.kick-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.kickPlayer(e.target.dataset.id);
            });
        });
    }

    kickPlayer(playerId) {
        if (this.socket && this.isHost) {
            this.socket.send(JSON.stringify({
                type: 'kick_player',
                player_id: playerId
            }));
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.game = new MafiaGame();
});
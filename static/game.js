class MafiaGame {
    constructor() {
        console.log("Game instance created");
        this.socket = null;
        this.gameId = null;
        this.playerId = null;
        this.playerName = null;
        this.playerRole = null;
        this.isHost = false;
        this.players = [];
        this.playerReady = false;
        this.gamePhase = 'lobby';
        this.dayNumber = 0;

        // Инициализация элементов DOM
        this.initElements();
        this.setupEventListeners();
    }

    initElements() {
        this.readyBtn = document.getElementById('ready-btn');
        this.startBtn = document.getElementById('start-game-btn');
        this.playersList = document.getElementById('players-list');
        this.createBtn = document.getElementById('create-game-btn');
        this.joinBtn = document.getElementById('join-game-btn');
        this.playerNameInput = document.getElementById('player-name-input');
        this.gameIdInput = document.getElementById('game-id-input');
        this.lobbyScreen = document.getElementById('lobby-screen');
        this.gameScreen = document.getElementById('game-screen');
        this.chatInput = document.getElementById('chat-input');
        this.chatSendBtn = document.getElementById('chat-send-btn');
        this.chatMessages = document.getElementById('chat-messages');
        this.dayPhaseInfo = document.getElementById('day-phase-info');
        this.voteButtons = document.getElementById('vote-buttons');
        this.nightActions = document.getElementById('night-actions');
        this.mafiaChat = document.getElementById('mafia-chat');
        this.roleInfo = document.getElementById('role-info');
    }

    setupEventListeners() {
        if (this.readyBtn) this.readyBtn.addEventListener('click', () => this.toggleReady());
        if (this.createBtn) this.createBtn.addEventListener('click', () => this.createGame());
        if (this.joinBtn) this.joinBtn.addEventListener('click', () => this.joinGame());
        if (this.startBtn) this.startBtn.addEventListener('click', () => this.startGame());
        if (this.chatSendBtn) this.chatSendBtn.addEventListener('click', () => this.sendChatMessage());
    }

    async createGame() {
        this.playerName = this.playerNameInput.value.trim();
        if (!this.playerName) {
            alert("Введите ваше имя");
            return;
        }

        try {
            const response = await fetch('/create', { method: 'POST' });
            const data = await response.json();
            this.gameId = data.game_id;
            this.connectWebSocket();
        } catch (error) {
            console.error('Create game error:', error);
        }
    }

    async joinGame() {
        this.playerName = this.playerNameInput.value.trim();
        this.gameId = this.gameIdInput.value.trim();
        
        if (!this.playerName || !this.gameId) {
            alert("Введите имя и ID игры");
            return;
        }

        try {
            const response = await fetch(`/join/${this.gameId}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ player_name: this.playerName })
            });
            this.connectWebSocket();
        } catch (error) {
            console.error('Join game error:', error);
        }
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        const wsUrl = `${protocol}${window.location.host}/ws?game_id=${this.gameId}&name=${encodeURIComponent(this.playerName)}`;
        
        console.log("Connecting to:", wsUrl);
        this.socket = new WebSocket(wsUrl);

        this.socket.onopen = () => {
            console.log("WebSocket connected");
            this.showLobby();
        };

        this.socket.onmessage = (event) => {
            console.log("Message received:", event.data);
            try {
                const msg = JSON.parse(event.data);
                this.handleMessage(msg);
            } catch (e) {
                console.error("Error parsing message:", e);
            }
        };

        this.socket.onerror = (error) => {
            console.error("WebSocket error:", error);
        };

        this.socket.onclose = () => {
            console.log("WebSocket disconnected");
        };
    }

    showLobby() {
        document.getElementById('lobby-screen').classList.remove('hidden');
        document.getElementById('game-screen').classList.add('hidden');
    }

    showGame() {
        document.getElementById('lobby-screen').classList.add('hidden');
        document.getElementById('game-screen').classList.remove('hidden');
    }

    handleMessage(msg) {
        switch(msg.type) {
            case 'players_update':
                this.updatePlayersList(msg.payload.players);
                break;
            case 'host_status':
                this.isHost = msg.payload.isHost;
                if (this.isHost && this.startBtn) {
                    this.startBtn.classList.remove('hidden');
                }
                break;
            case 'role_assigned':
                this.playerRole = msg.payload.role;
                this.showRoleInfo();
                break;
            case 'night_start':
                this.startNightPhase(msg.payload.number);
                break;
            case 'day_start':
                this.startDayPhase(msg.payload.number, msg.payload.killed);
                break;
            case 'game_end':
                this.endGame(msg.payload.winner);
                break;
            case 'player_killed':
                this.showPlayerKilled(msg.payload.player_id);
                break;
            case 'chat':
                this.addChatMessage(msg.payload.sender, msg.payload.text);
                break;
            default:
                console.log("Unknown message type:", msg.type);
        }
    }

    showRoleInfo() {
        const roleNames = {
            'don': 'Дон мафии',
            'mafia': 'Мафия',
            'sheriff': 'Шериф',
            'civilian': 'Мирный житель'
        };
        
        this.roleInfo.innerHTML = `Ваша роль: <strong>${roleNames[this.playerRole]}</strong>`;
        
        if (this.playerRole === 'don' || this.playerRole === 'mafia') {
            this.mafiaChat.classList.remove('hidden');
        }
    }

    startNightPhase(dayNumber) {
        this.gamePhase = 'night';
        this.dayNumber = dayNumber;
        this.dayPhaseInfo.innerHTML = `Ночь ${dayNumber}`;
        
        // Показать соответствующие действия для роли
        if (this.playerRole === 'don') {
            this.nightActions.innerHTML = `
                <h3>Выберите игрока для проверки на шерифа:</h3>
                ${this.generatePlayerSelection('check')}
            `;
        } else if (this.playerRole === 'sheriff') {
            this.nightActions.innerHTML = `
                <h3>Выберите игрока для проверки на мафию:</h3>
                ${this.generatePlayerSelection('check')}
            `;
        } else if (this.playerRole === 'mafia') {
            this.nightActions.innerHTML = `
                <h3>Выберите игрока для убийства:</h3>
                ${this.generatePlayerSelection('kill')}
            `;
        } else {
            this.nightActions.innerHTML = '<p>Вы мирный житель. Спите.</p>';
        }
    }

    startDayPhase(dayNumber, killedPlayerId) {
        this.gamePhase = 'day';
        this.dayNumber = dayNumber;
        
        let killedText = '';
        if (killedPlayerId) {
            const player = this.players.find(p => p.id === killedPlayerId);
            killedText = player ? `Ночью был убит ${player.name}.` : 'Ночью никто не был убит.';
        } else {
            killedText = 'Ночью никто не был убит.';
        }
        
        this.dayPhaseInfo.innerHTML = `День ${dayNumber}<br>${killedText}`;
        this.nightActions.innerHTML = '';
        
        // Показать кнопки для голосования
        this.voteButtons.innerHTML = `
            <h3>Голосование:</h3>
            ${this.generatePlayerSelection('vote')}
            <button id="abstain-vote">Воздержаться</button>
        `;
        
        document.getElementById('abstain-vote').addEventListener('click', () => {
            this.sendVote('');
        });
    }

    generatePlayerSelection(actionType) {
        return this.players
            .filter(p => p.alive && p.id !== this.playerId)
            .map(p => `
                <button class="player-select" data-id="${p.id}" data-action="${actionType}">
                    ${p.name}
                </button>
            `)
            .join('');
    }

    sendVote(targetId) {
        if (this.socket) {
            this.socket.send(JSON.stringify({
                type: 'vote',
                target: targetId
            }));
        }
    }

    sendNightAction(targetId, actionType) {
        if (this.socket) {
            this.socket.send(JSON.stringify({
                type: 'night_action',
                action: actionType,
                target: targetId
            }));
        }
    }

    sendChatMessage() {
        const message = this.chatInput.value.trim();
        if (message && this.socket) {
            this.socket.send(JSON.stringify({
                type: 'chat',
                message: message
            }));
            this.chatInput.value = '';
        }
    }

    addChatMessage(sender, text) {
        const messageElement = document.createElement('div');
        messageElement.className = 'chat-message';
        messageElement.innerHTML = `<strong>${sender}:</strong> ${text}`;
        this.chatMessages.appendChild(messageElement);
        this.chatMessages.scrollTop = this.chatMessages.scrollHeight;
    }

    endGame(winner) {
        const winnerNames = {
            'mafia': 'Мафия',
            'civilians': 'Мирные жители'
        };
        
        this.dayPhaseInfo.innerHTML = `Игра окончена! Победили ${winnerNames[winner]}`;
        this.voteButtons.innerHTML = '';
        this.nightActions.innerHTML = '';
        
        // Показать все роли
        const rolesInfo = this.players.map(p => 
            `${p.name}: ${p.role}`
        ).join('<br>');
        
        this.nightActions.innerHTML = `<h3>Роли всех игроков:</h3>${rolesInfo}`;
    }

    // ... остальные методы из предыдущей версии ...
}
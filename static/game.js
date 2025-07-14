class MafiaGame {
    constructor() {
        console.log("Game instance created");
        this.socket = null;
        this.gameId = null;
        this.playerName = null;
        this.isHost = false;
        this.players = [];
        this.playerReady = false;

        // Инициализация элементов DOM
        this.readyBtn = document.getElementById('ready-btn');
        this.startBtn = document.getElementById('start-game-btn');
        this.playersList = document.getElementById('players-list');
        this.createBtn = document.getElementById('create-game-btn');
        this.joinBtn = document.getElementById('join-game-btn');
        this.playerNameInput = document.getElementById('player-name-input');
        this.gameIdInput = document.getElementById('game-id-input');

        // Проверка элементов
        if (!this.readyBtn || !this.playersList) {
            console.error("Critical DOM elements not found!");
            return;
        }

        // Назначаем обработчики событий
        this.readyBtn.addEventListener('click', () => this.toggleReady());
        if (this.createBtn) this.createBtn.addEventListener('click', () => this.createGame());
        if (this.joinBtn) this.joinBtn.addEventListener('click', () => this.joinGame());
        if (this.startBtn) this.startBtn.addEventListener('click', () => this.startGame());
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
            default:
                console.log("Unknown message type:", msg.type);
        }
    }

    updatePlayersList(players) {
        console.log("Updating players list:", players);
        this.players = players;
        this.playersList.innerHTML = '';

        players.forEach(player => {
            const playerEl = document.createElement('div');
            playerEl.className = `player ${player.ready ? 'ready' : ''}`;
            playerEl.innerHTML = `
                <span class="player-name">${player.name}</span>
                <span class="player-status">${player.ready ? '✓ Готов' : 'Не готов'}</span>
            `;
            this.playersList.appendChild(playerEl);
        });
    }

    toggleReady() {
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            console.error("WebSocket not ready");
            return;
        }

        this.playerReady = !this.playerReady;
        this.readyBtn.textContent = this.playerReady ? '✓ Готов' : 'Готов';
        this.readyBtn.classList.toggle('ready', this.playerReady);

        this.socket.send(JSON.stringify({
            type: 'set_ready',
            ready: this.playerReady
        }));
    }

    startGame() {
        if (this.isHost && this.socket) {
            this.socket.send(JSON.stringify({
                type: 'start_game'
            }));
        }
    }
}

// Инициализация игры после загрузки DOM
document.addEventListener('DOMContentLoaded', () => {
    window.game = new MafiaGame();
});
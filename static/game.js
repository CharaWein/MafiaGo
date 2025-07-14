class MafiaGame {
    constructor() {
        console.log("Game instance created");
        this.socket = null;
        this.gameId = null;
        this.playerName = null;
        this.isHost = false;
        this.players = [];
        this.playerId = null; // Добавляем хранение ID игрока
        this.playerReady = false;

        this.initEventListeners();
    }

    async createGame() {
        this.playerName = document.getElementById('player-name-input').value.trim();
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
        this.playerName = document.getElementById('player-name-input').value.trim();
        this.gameId = document.getElementById('game-id-input').value.trim();
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
        const wsUrl = `ws://${window.location.host}/ws?game_id=${this.gameId}&name=${encodeURIComponent(this.playerName)}`;
        this.socket = new WebSocket(wsUrl);

        this.socket.onopen = () => {
            console.log('WebSocket connected');
        };

        this.socket.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            console.log('Received:', msg);

            if (msg.type === 'init') {
                this.playerId = msg.payload.id;
                this.updatePlayersList(msg.payload.players);
            }
            else if (msg.type === 'players_update') {
                this.updatePlayersList(msg.payload.players);
            }
            else if (msg.type === 'host_status') {
                this.isHost = msg.payload.isHost;
            }
        };
    }

    updatePlayersList(players) {
        console.log('Updating players:', players);
        this.players = players;
        const listElement = document.getElementById('players-list');
        listElement.innerHTML = '';

        players.forEach(player => {
            const playerElement = document.createElement('div');
            playerElement.className = `player ${player.ready ? 'ready' : ''}`;
            playerElement.innerHTML = `
                <span>${player.name}</span>
                <span>${player.ready ? '✓ Готов' : 'Не готов'}</span>
            `;
            listElement.appendChild(playerElement);
        });
    }

    toggleReady() {
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            console.error('WebSocket not ready');
            return;
        }

        this.playerReady = !this.playerReady;
        const readyBtn = document.getElementById('ready-btn');
        readyBtn.textContent = this.playerReady ? '✓ Готов' : 'Готов';
        readyBtn.classList.toggle('ready', this.playerReady);

        this.socket.send(JSON.stringify({
            type: 'set_ready',
            ready: this.playerReady
        }));
    }
}

window.game = new MafiaGame();
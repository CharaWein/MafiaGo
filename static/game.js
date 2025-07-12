class MafiaGame {
    constructor() {
        this.socket = null;
        this.gameId = null;
        this.playerName = null;
        this.isHost = false;
        this.players = [];
        this.playerReady = false;

        console.log("Game instance created"); // Добавляем лог создания
    
        // Проверяем элементы DOM
        console.log("Ready button exists:", !!document.getElementById('ready-btn'));
        console.log("Players list exists:", !!document.getElementById('players-list'));
        
        this.initEventListeners();
        console.log("Game initialized"); // Добавим лог инициализации
    }

    initEventListeners() {
        document.getElementById('create-game-btn').addEventListener('click', () => this.createGame());
        document.getElementById('join-game-btn').addEventListener('click', () => this.joinGame());
        
        const readyBtn = document.getElementById('ready-btn');
        readyBtn.addEventListener('click', () => this.toggleReady());
        
        const startBtn = document.getElementById('start-game-btn');
        if (startBtn) {
            startBtn.addEventListener('click', () => this.startGame());
        }
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        const wsUrl = `${protocol}${window.location.host}/ws?game_id=${this.gameId}&name=${encodeURIComponent(this.playerName)}`;
        
        console.log("Connecting to WebSocket:", wsUrl); // Лог URL подключения
        
        this.socket = new WebSocket(wsUrl);

        this.socket.onopen = () => {
            console.log('WebSocket connection established');
            this.socket.send(JSON.stringify({
                type: 'get_players' // Запрашиваем текущий список игроков
            }));
        };

        this.socket.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.socket.onmessage = (event) => {
            console.log('Raw WebSocket message:', event.data);
            try {
                const msg = JSON.parse(event.data);
                console.log('Parsed WebSocket message:', msg);
                
                switch(msg.type) {
                    case 'players_update':
                        console.log('Received players update:', msg.payload);
                        this.updatePlayersList(msg.payload.players);
                        this.updateStartButton(msg.payload.canStart);
                        break;
                    case 'host_status':
                        console.log('Host status:', msg.payload.isHost);
                        this.isHost = msg.payload.isHost;
                        if (this.isHost) {
                            document.getElementById('start-game-btn').classList.remove('hidden');
                        }
                        break;
                    default:
                        console.log('Unknown message type:', msg.type);
                }
            } catch (e) {
                console.error('Error parsing WebSocket message:', e);
            }
        };
    }

    updatePlayersList(players) {
        console.log("Updating players list with:", players);
        this.players = players;
        const listElement = document.getElementById('players-list');
        if (!listElement) {
            console.error("Players list element not found!");
            return;
        }
        
        listElement.innerHTML = '';
        
        players.forEach(player => {
            const playerElement = document.createElement('div');
            playerElement.className = `player ${player.ready ? 'ready' : ''}`;
            playerElement.innerHTML = `
                <span class="player-name">${player.name}</span>
                <span class="player-status">
                    ${player.ready ? '✓ Готов' : 'Не готов'}
                </span>
            `;
            listElement.appendChild(playerElement);
        });
    }

    toggleReady() {
        console.log("Toggle ready called");
        if (!this.socket) {
            console.error("WebSocket is not initialized");
            return;
        }
        
        if (this.socket.readyState !== WebSocket.OPEN) {
            console.error("WebSocket is not in OPEN state. Current state:", this.socket.readyState);
            return;
        }
        
        this.playerReady = !this.playerReady;
        const readyBtn = document.getElementById('ready-btn');
        readyBtn.textContent = this.playerReady ? '✓ Готов' : 'Готов';
        readyBtn.className = this.playerReady ? 'ready' : '';
        
        const message = {
            type: 'set_ready',
            ready: this.playerReady
        };
        
        console.log("Sending ready status:", message);
        this.socket.send(JSON.stringify(message));
    }
    
    initEventListeners() {
        document.getElementById('create-game-btn').addEventListener('click', () => this.createGame());
        document.getElementById('join-game-btn').addEventListener('click', () => this.joinGame());
        document.getElementById('ready-btn').addEventListener('click', () => this.toggleReady());
        document.getElementById('start-game-btn').addEventListener('click', () => this.startGame());
    }

    async createGame() {
        this.playerName = document.getElementById('player-name-input').value.trim();
        if (!this.playerName) {
            alert('Введите ваше имя');
            return;
        }

        try {
            const response = await fetch('/create', { method: 'POST' });
            const data = await response.json();
            
            this.gameId = data.game_id;
            this.showLobby();
            this.connectWebSocket();
        } catch (error) {
            console.error('Ошибка при создании игры:', error);
        }
    }
    
    async joinGame() {
        this.playerName = document.getElementById('player-name-input').value.trim();
        this.gameId = document.getElementById('game-id-input').value.trim();
        
        if (!this.playerName || !this.gameId) {
            alert('Введите имя и ID игры');
            return;
        }

        try {
            const response = await fetch(`/join/${this.gameId}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ player_name: this.playerName })
            });
            
            if (!response.ok) {
                throw new Error(await response.text());
            }
            
            this.showLobby();
            this.connectWebSocket();
        } catch (error) {
            alert(`Ошибка: ${error.message}`);
        }
    }

    showLobby() {
        document.getElementById('start-screen').classList.add('hidden');
        document.getElementById('lobby-screen').classList.remove('hidden');
        document.getElementById('game-id-display').textContent = this.gameId;
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
        const wsUrl = `${protocol}${window.location.host}/ws?game_id=${this.gameId}&name=${encodeURIComponent(this.playerName)}`;
        
        this.socket = new WebSocket(wsUrl);

        this.socket.onmessage = (event) => {
            const msg = JSON.parse(event.data);
            console.log('WebSocket message:', msg);
            
            switch(msg.type) {
                case 'lobby_state':
                    this.updatePlayersList(msg.payload.players);
                    this.updateStartButton(msg.payload.canStart);
                    break;
                case 'host_status':
                    this.isHost = msg.payload.isHost;
                    if (this.isHost) {
                        document.getElementById('start-game-btn').classList.remove('hidden');
                    }
                    break;
                case 'game_started':
                    this.startGame();
                    break;
            }
        };
    }

    toggleReady() {
        if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
            console.error("WebSocket is not connected");
            return;
        }

        this.playerReady = !this.playerReady;
        const readyBtn = document.getElementById('ready-btn');
        
        // Визуальное обновление
        readyBtn.textContent = this.playerReady ? '✓ Готов' : 'Готов';
        readyBtn.classList.toggle('active', this.playerReady);
        
        // Отправка состояния на сервер
        const message = {
            type: 'set_ready',
            ready: this.playerReady
        };
        
        console.log("Sending ready status:", message);
        this.socket.send(JSON.stringify(message));

    }

    updatePlayersList(players) {
        console.log("Received players list:", players);
        this.players = players;
        const listElement = document.getElementById('players-list');
        listElement.innerHTML = '';
        
        players.forEach(player => {
            const playerElement = document.createElement('div');
            playerElement.className = `player ${player.ready ? 'ready' : ''}`;
            playerElement.dataset.playerId = player.id;
            
            playerElement.innerHTML = `
                <span class="player-name">${player.name}</span>
                <span class="player-status">
                    ${player.ready ? '✓ Готов' : 'Не готов'}
                </span>
            `;
            listElement.appendChild(playerElement);
        });
    }

    toggleReady() {
        if (this.socket) {
            const isReady = !document.getElementById('ready-btn').textContent.includes('✓');
            this.socket.send(JSON.stringify({
                type: 'set_ready',
                ready: isReady
            }));
            
            document.getElementById('ready-btn').textContent = isReady ? '✓ Готов' : 'Готов';
        }
    }

    updateStartButton(canStart) {
        const startBtn = document.getElementById('start-game-btn');
        if (this.isHost) {
            startBtn.disabled = !canStart;
        }
    }

    toggleReady() {
        if (this.socket) {
            const isReady = document.getElementById('ready-btn').textContent.includes('✓');
            this.socket.send(JSON.stringify({
                type: 'set_ready',
                payload: !isReady
            }));
            
            document.getElementById('ready-btn').textContent = 
                isReady ? 'Готов' : '✓ Готов';
        }
    }

    startGame() {
        if (this.socket && this.isHost) {
            this.socket.send(JSON.stringify({
                type: 'start_game'
            }));
        }
    }

    showGameScreen() {
        document.getElementById('lobby-screen').classList.add('hidden');
        document.getElementById('game-screen').classList.remove('hidden');
    }

    initReadyButton() {
        const readyBtn = document.getElementById('ready-btn');
        readyBtn.addEventListener('click', () => {
            if (!this.socket) {
                console.error("WebSocket not connected");
                return;
            }

            const isReady = !readyBtn.classList.contains('ready');
            
            // Визуальное обновление
            readyBtn.classList.toggle('ready', isReady);
            readyBtn.textContent = isReady ? '✓ Готов' : 'Готов';
            
            // Отправка на сервер
            this.socket.send(JSON.stringify({
                type: 'set_ready',
                payload: isReady
            }));
            
            console.log("Sent ready status:", isReady);
        });
    }

    updatePlayersList(players) {
        console.log("Full players update:", players);
        const listElement = document.getElementById('players-list');
        listElement.innerHTML = '';
        
        players.forEach(player => {
            const playerEl = document.createElement('div');
            playerEl.className = `player ${player.ready ? 'ready' : ''}`;
            playerEl.innerHTML = `
                <span class="player-name">${player.name}</span>
                <span class="player-status">
                    ${player.ready ? '✓ Готов' : 'Не готов'}
                </span>
            `;
            listElement.appendChild(playerEl);
        });
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.game = new MafiaGame();
});
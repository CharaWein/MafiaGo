class MafiaGame {
    constructor() {
        this.socket = null;
        this.gameId = null;
        this.playerId = null;
        this.playerName = null;
        
        this.initEventListeners();
    }
    
    initEventListeners() {
        document.getElementById('create-game-btn').addEventListener('click', () => this.createGame());
        document.getElementById('join-game-btn').addEventListener('click', () => this.joinGame());
        document.getElementById('send-message-btn').addEventListener('click', () => this.sendChatMessage());
    }
    
    async createGame() {
        try {
            const response = await fetch('/create', { method: 'POST' });
            const data = await response.json();
            
            this.gameId = data.game_id;
            document.getElementById('game-id-display').textContent = this.gameId;
            this.showMessage(`Игра создана! ID: ${this.gameId}`);
            
            // Переходим к экрану ввода имени
            document.getElementById('game-id-input').value = this.gameId;
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
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ player_name: this.playerName })
            });
            
            const data = await response.json();
            
            if (data.status === 'joined') {
                this.connectWebSocket();
                document.getElementById('lobby-screen').style.display = 'none';
                document.getElementById('game-screen').style.display = 'block';
                this.showMessage(`Вы присоединились к игре как ${this.playerName}`);
            } else {
                this.showMessage('Ошибка при присоединении к игре');
            }
        } catch (error) {
            this.showMessage(`Ошибка: ${error}`);
        }
    }
    
    connectWebSocket() {
		const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
		const wsUrl = `${protocol}${window.location.host}/ws?game_id=${this.gameId}&name=${encodeURIComponent(this.playerName)}`;
		
		this.socket = new WebSocket(wsUrl);

		this.socket.onmessage = (event) => {
			const msg = JSON.parse(event.data);
			switch(msg.type) {
				case 'game_state':
					this.updateGameState(msg.payload);
					break;
				case 'role_assigned':
					this.handleRole(msg.role);
					break;
				case 'night_start':
					this.showNightInterface();
					break;
				case 'day_start':
					this.showVotingInterface();
					break;
				case 'player_killed':
					this.showKilledPlayer(msg.payload);
					break;
			}
		};
	}
    
    handleGameMessage(message) {
        switch(message.type) {
            case 'game_state':
                this.updateGameState(message.payload);
                break;
            case 'chat_message':
                this.addChatMessage(message.payload);
                break;
            default:
                console.log('Неизвестный тип сообщения:', message);
        }
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
                <div>${player.name}</div>
                <div>${player.alive ? 'Жив' : 'Мертв'}</div>
            `;
            playersList.appendChild(playerCard);
        });
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
        messageElement.textContent = `${message.sender}: ${message.text}`;
        chat.appendChild(messageElement);
        chat.scrollTop = chat.scrollHeight;
    }
    
    showMessage(text) {
        console.log(text);
        // Можно добавить отображение сообщений в UI
    }

    handleGameMessage(message) {
        switch (message.type) {
            case 'game_state':
                this.updateGameState(message.payload);
                break;
            case 'role_info':
                this.showRole(message.payload);
                break;
            case 'night_start':
                this.showNightActions();
                break;
            case 'day_start':
                this.showVoting();
                break;
            case 'game_end':
                this.showGameEnd(message.payload);
                break;
        }
    }
    
    showNightActions() {
        if (this.role === 'mafia' || this.role === 'don') {
            this.showPlayerSelector('Выберите цель для ночного действия:', players => {
                this.socket.send(JSON.stringify({
                    type: 'night_action',
                    target: players[0].id
                }));
            });
        }
    }
    
    showVoting() {
        this.showPlayerSelector('Голосуйте, кого казнить:', players => {
            this.socket.send(JSON.stringify({
                type: 'vote',
                target: players[0].id
            }));
        });
    }

    handleChatMessage(message) {
        const chat = document.getElementById('chat-messages');
        const msgElement = document.createElement('div');
        msgElement.innerHTML = `
            <strong>${message.sender}</strong> 
            [${message.time}]: ${message.text}
        `;
        chat.appendChild(msgElement);
        chat.scrollTop = chat.scrollHeight;
    }

    sendChatMessage() {
        const input = document.getElementById('chat-input');
        const text = input.value.trim();
        
        if (text && this.socket) {
            this.socket.send(JSON.stringify({
                type: 'chat',
                payload: text
            }));
            input.value = '';
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
}

// Инициализация игры при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
    window.game = new MafiaGame();
});
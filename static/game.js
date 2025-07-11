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
        this.socket = new WebSocket(`ws://${window.location.host}/ws/${this.gameId}?player=${this.playerName}`);
        
        this.socket.onopen = () => {
            this.showMessage('WebSocket соединение установлено');
        };
        
        this.socket.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleGameMessage(message);
        };
        
        this.socket.onclose = () => {
            this.showMessage('Соединение закрыто');
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
}

// Инициализация игры при загрузке страницы
document.addEventListener('DOMContentLoaded', () => {
    window.game = new MafiaGame();
});
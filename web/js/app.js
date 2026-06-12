class XSMApp {
    constructor() {
        this.token = localStorage.getItem('xsm_token');
        this.currentUser = null;
        this.currentChat = null;
        this.chats = [];
        this.messages = [];
        this.init();
    }

    init() {
        this.cacheDOM();
        this.bindEvents();
        if (this.token) {
            this.showScreen('chat-screen');
            this.loadUserInfo();
            this.loadChats();
        } else {
            this.showScreen('auth-screen');
        }
    }

    cacheDOM() {
        this.authScreen = document.getElementById('auth-screen');
        this.loginForm = document.getElementById('login-form');
        this.registerForm = document.getElementById('register-form');
        this.loginTab = document.getElementById('login-tab');
        this.registerTab = document.getElementById('register-tab');
        this.loginError = document.getElementById('login-error');
        this.registerError = document.getElementById('register-error');
        this.chatScreen = document.getElementById('chat-screen');
        this.chatsList = document.getElementById('chats-list');
        this.messagesList = document.getElementById('messages-list');
        this.messagesContainer = document.getElementById('messages-container');
        this.messageInput = document.getElementById('message-input');
        this.sendBtn = document.getElementById('send-btn');
        this.logoutBtn = document.getElementById('logout-btn');
        this.newChatBtn = document.getElementById('new-chat-btn');
        this.currentUsername = document.getElementById('current-username');
        this.currentUserAvatar = document.getElementById('current-user-avatar');
        this.chatName = document.getElementById('chat-name');
        this.chatStatus = document.getElementById('chat-status');
        this.noChat = document.getElementById('no-chat');
        this.chatView = document.getElementById('chat-view');
        this.searchInput = document.getElementById('search-input');
        this.newChatModal = document.getElementById('new-chat-modal');
        this.newChatUsername = document.getElementById('new-chat-username');
        this.createChatBtn = document.getElementById('create-chat-btn');
        this.cancelChatBtn = document.getElementById('cancel-chat-btn');
        this.emojiBtn = document.getElementById('emoji-btn');
        this.emojiPicker = document.getElementById('emoji-picker');
    }

    bindEvents() {
        this.loginTab.addEventListener('click', () => this.switchAuthTab('login'));
        this.registerTab.addEventListener('click', () => this.switchAuthTab('register'));
        this.loginForm.addEventListener('submit', (e) => { e.preventDefault(); this.login(); });
        this.registerForm.addEventListener('submit', (e) => { e.preventDefault(); this.register(); });
        this.logoutBtn.addEventListener('click', () => this.logout());
        this.sendBtn.addEventListener('click', () => this.sendMessage());
        this.messageInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); this.sendMessage(); }
        });
        this.messageInput.addEventListener('input', () => this.autoResize());
        this.newChatBtn.addEventListener('click', () => this.openNewChatModal());
        this.cancelChatBtn.addEventListener('click', () => this.closeNewChatModal());
        this.createChatBtn.addEventListener('click', () => this.createChat());
        this.newChatModal.querySelector('.modal-backdrop').addEventListener('click', () => this.closeNewChatModal());
        this.searchInput.addEventListener('input', (e) => this.filterChats(e.target.value));
        this.emojiBtn.addEventListener('click', () => this.toggleEmojiPicker());
        document.addEventListener('click', (e) => {
            if (!this.emojiPicker.contains(e.target) && e.target !== this.emojiBtn) {
                this.emojiPicker.style.display = 'none';
            }
        });
        this.emojiPicker.querySelectorAll('span').forEach(emoji => {
            emoji.addEventListener('click', () => this.insertEmoji(emoji.textContent));
        });
    }

    showScreen(id) {
        document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
        document.getElementById(id).classList.add('active');
    }

    switchAuthTab(tab) {
        const isLogin = tab === 'login';
        this.loginTab.classList.toggle('active', isLogin);
        this.registerTab.classList.toggle('active', !isLogin);
        this.loginForm.classList.toggle('active', isLogin);
        this.registerForm.classList.toggle('active', !isLogin);
    }

    async apiCall(url, options = {}) {
        const headers = { 'Authorization': `Bearer ${this.token}`, ...options.headers };
        const res = await fetch(url, { ...options, headers });
        if (res.status === 401) { this.logout(); throw new Error('Unauthorized'); }
        return res;
    }

    async login() {
        const username = document.getElementById('login-username').value.trim();
        const password = document.getElementById('login-password').value;
        if (!username || !password) return;
        try {
            const res = await fetch('/api/v1/auth/login', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || 'Ошибка входа');
            this.token = data.access_token;
            localStorage.setItem('xsm_token', this.token);
            this.showScreen('chat-screen');
            this.loadUserInfo();
            this.loadChats();
        } catch (err) { this.loginError.textContent = err.message; }
    }

    async register() {
        const username = document.getElementById('reg-username').value.trim();
        const email = document.getElementById('reg-email').value.trim();
        const password = document.getElementById('reg-password').value;
        if (!username || !email || !password) return;
        try {
            const res = await fetch('/api/v1/auth/register', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, email, password })
            });
            const data = await res.json();
            if (!res.ok) throw new Error(data.error || 'Ошибка регистрации');
            this.registerError.textContent = '';
            this.switchAuthTab('login');
            alert('Регистрация успешна! Теперь войдите.');
            document.getElementById('login-username').value = username;
        } catch (err) { this.registerError.textContent = err.message; }
    }

    logout() {
        localStorage.removeItem('xsm_token');
        this.token = null; this.currentUser = null; this.chats = []; this.messages = [];
        this.showScreen('auth-screen');
    }

    async loadUserInfo() {
        try {
            const res = await this.apiCall('/api/v1/user/profile');
            const data = await res.json();
            this.currentUser = data;
            this.currentUsername.textContent = data.username || 'User';
            this.currentUserAvatar.textContent = (data.username || 'U')[0].toUpperCase();
        } catch (e) { console.error('Ошибка загрузки профиля'); }
    }

    async loadChats() {
        try {
            const res = await this.apiCall('/api/v1/chats');
            const data = await res.json();
            this.chats = data.chats || [];
            this.renderChats();
        } catch (e) { console.error('Ошибка загрузки чатов'); }
    }

    renderChats(filter = '') {
        const filtered = this.chats.filter(chat =>
            (chat.name || 'Чат').toLowerCase().includes(filter.toLowerCase())
        );
        this.chatsList.innerHTML = '';
        filtered.forEach(chat => {
            const div = document.createElement('div');
            div.className = `chat-item ${this.currentChat?.id === chat.id ? 'active' : ''}`;
            div.innerHTML = `
                <div class="chat-avatar">${(chat.name || 'Ч')[0]}</div>
                <div class="chat-info">
                    <div class="chat-name">${chat.name || 'Чат'}</div>
                    <div class="chat-preview">${chat.is_group ? 'Группа' : 'Личный'}</div>
                </div>
                <div class="chat-meta">
                    <span class="chat-time">${new Date(chat.updated_at).toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'})}</span>
                </div>
            `;
            div.addEventListener('click', () => this.selectChat(chat));
            this.chatsList.appendChild(div);
        });
    }

    filterChats(query) { this.renderChats(query); }

    selectChat(chat) {
        this.currentChat = chat;
        this.chatName.textContent = chat.name || 'Чат';
        this.chatStatus.textContent = chat.is_group ? 'Групповой чат' : 'Личный чат';
        this.noChat.style.display = 'none';
        this.chatView.style.display = 'flex';
        this.messageInput.focus();
        this.renderChats();
        this.loadMessages(chat.id);
    }

    async loadMessages(chatId) {
        try {
            const res = await this.apiCall(`/api/v1/messages/${chatId}`);
            const data = await res.json();
            this.messages = data.messages || [];
            this.renderMessages();
        } catch (e) { console.error('Ошибка загрузки сообщений'); }
    }

    renderMessages() {
        this.messagesList.innerHTML = '';
        this.messages.forEach(msg => {
            const isSent = msg.sender_id === this.currentUser?.id;
            const div = document.createElement('div');
            div.className = `message ${isSent ? 'sent' : 'received'}`;
            const content = msg.encrypted_content || '🔒 Зашифровано';
            div.innerHTML = `
                <div class="message-content">${this.escapeHtml(content)}</div>
                <div class="message-meta">
                    <span class="message-time">${new Date(msg.created_at).toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'})}</span>
                </div>
            `;
            this.messagesList.appendChild(div);
        });
        this.scrollToBottom();
    }

    async sendMessage() {
        const content = this.messageInput.value.trim();
        if (!content || !this.currentChat) return;
        const payload = {
            chat_id: this.currentChat.id,
            encrypted_content: content,
            content_type: 'text',
            iv: new Array(16).fill(0)
        };
        try {
            const res = await this.apiCall('/api/v1/messages', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (res.ok) {
                this.messageInput.value = '';
                this.autoResize();
                await this.loadMessages(this.currentChat.id);
            }
        } catch (e) { console.error('Ошибка отправки'); }
    }

    autoResize() {
        const el = this.messageInput;
        el.style.height = 'auto';
        el.style.height = Math.min(el.scrollHeight, 120) + 'px';
    }

    scrollToBottom() {
        if (this.messagesContainer) this.messagesContainer.scrollTop = this.messagesContainer.scrollHeight;
    }

    escapeHtml(text) {
        const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' };
        return text.replace(/[&<>"']/g, m => map[m]);
    }

    openNewChatModal() { this.newChatModal.classList.add('active'); this.newChatUsername.focus(); }
    closeNewChatModal() { this.newChatModal.classList.remove('active'); this.newChatUsername.value = ''; }

    async createChat() {
        const username = this.newChatUsername.value.trim();
        if (!username) return;
        try {
            const res = await this.apiCall('/api/v1/chats', {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ user_id: this.currentUser?.id, is_group: false, participants: [username] })
            });
            if (res.ok) { this.closeNewChatModal(); await this.loadChats(); }
            else { const err = await res.json(); alert(err.error || 'Ошибка'); }
        } catch (e) { console.error(e); }
    }

    toggleEmojiPicker() {
        this.emojiPicker.style.display = this.emojiPicker.style.display === 'none' ? 'block' : 'none';
    }

    insertEmoji(emoji) {
        const textarea = this.messageInput;
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        textarea.value = textarea.value.substring(0, start) + emoji + textarea.value.substring(end);
        textarea.focus();
        textarea.selectionStart = start + emoji.length;
        textarea.selectionEnd = start + emoji.length;
        this.emojiPicker.style.display = 'none';
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.xsmApp = new XSMApp();
});
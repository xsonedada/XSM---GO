// ==========================================
// XSM MESSENGER – Полный JavaScript (исправлено)
// ==========================================

document.addEventListener('DOMContentLoaded', () => {
    const token = localStorage.getItem('xsm_token');
    if (!token) { window.location.href = '/web/auth.html'; return; }

    // ===== ГЛОБАЛЬНЫЕ ПЕРЕМЕННЫЕ =====
    let currentUser = null, currentChat = null, chats = [], messages = [];
    let ws = null, reconnectTimer = null, contextMenu = null, replyToMessage = null;
    let unreadCounts = {}, totalUnread = 0, isPageActive = true, notificationPermission = 'default';
    let selectedUser = null;

    // ===== DOM ЭЛЕМЕНТЫ =====
    const navAvatar = document.getElementById('nav-avatar');
    const chatsList = document.getElementById('chats-list');
    const searchInput = document.getElementById('search-input');
    const newChatBtn = document.getElementById('new-chat-btn');
    const emptyState = document.getElementById('empty-state');
    const chatView = document.getElementById('chat-view');
    const partnerName = document.getElementById('partner-name');
    const partnerAvatar = document.getElementById('partner-avatar');
    const partnerStatus = document.getElementById('partner-status');
    const messagesContainer = document.getElementById('messages-container');
    const messagesList = document.getElementById('messages-list');
    const messageInput = document.getElementById('message-input');
    const sendBtn = document.getElementById('send-btn');
    const emojiBtn = document.getElementById('emoji-btn');
    const emojiPicker = document.getElementById('emoji-picker');
    const newChatModal = document.getElementById('new-chat-modal');
    const newChatInput = document.getElementById('new-chat-input');
    const searchResults = document.getElementById('search-results');
    const createChatBtn = document.getElementById('create-chat-btn');
    const logoutBtn = document.getElementById('logout-btn');
    const saveProfileBtn = document.getElementById('save-profile-btn');
    const avatarUploadInput = document.getElementById('avatar-upload-input');
    const settingsStatusText = document.getElementById('settings-status-text');
    const settingsBio = document.getElementById('settings-bio');
    const editUsername = document.getElementById('edit-username');
    const editEmail = document.getElementById('edit-email');
    const settingsAvatarPreview = document.getElementById('settings-avatar-preview');
    const settingsUsernameDisplay = document.getElementById('settings-username-display');
    const settingsEmailDisplay = document.getElementById('settings-email-display');
    const scrollBottomBtn = document.getElementById('scroll-bottom-btn');

    // ===== КНОПКА «ПРОКРУТИТЬ ВНИЗ» =====
    if (messagesContainer && scrollBottomBtn) {
        messagesContainer.addEventListener('scroll', () => {
            const atBottom = messagesContainer.scrollHeight - messagesContainer.scrollTop - messagesContainer.clientHeight < 50;
            scrollBottomBtn.classList.toggle('visible', !atBottom);
        });
        scrollBottomBtn.addEventListener('click', () => {
            messagesContainer.scrollTo({ top: messagesContainer.scrollHeight, behavior: 'smooth' });
        });
    }

    // ===== API КЛИЕНТ =====
    async function apiCall(url, options = {}) {
        const headers = { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json', ...options.headers };
        const res = await fetch(url, { ...options, headers });
        if (res.status === 401) { localStorage.removeItem('xsm_token'); window.location.href = '/web/auth.html'; throw new Error('Unauthorized'); }
        return res;
    }

    // ===== WEB SOCKET =====
    function connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/api/v1/ws?token=${token}`;
        ws = new WebSocket(wsUrl);
        ws.onopen = () => { console.log('🟢 WS online'); if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; } };
        ws.onmessage = (event) => { try { handleWebSocketMessage(JSON.parse(event.data)); } catch(e) { console.error(e); } };
        ws.onclose = () => { console.log('🔴 WS offline, reconnect...'); reconnectTimer = setTimeout(connectWebSocket, 3000); };
        ws.onerror = (e) => console.error('WS error', e);
    }

    function handleWebSocketMessage(msg) {
        switch (msg.type) {
            case 'new_message':
                if (!currentChat || msg.payload.chat_id !== currentChat.id) {
                    unreadCounts[msg.payload.chat_id] = (unreadCounts[msg.payload.chat_id] || 0) + 1;
                    totalUnread++;
                    updateUnreadBadges();
                    const chat = chats.find(c => c.id === msg.payload.chat_id);
                    sendNotification(chat?.name || 'Новое сообщение', (msg.payload.encrypted_content || '').substring(0, 100));
                }
                if (currentChat && msg.payload.chat_id === currentChat.id) {
                    messages.push(msg.payload);
                    renderMessages();
                    if (isPageActive) markChatAsRead(currentChat.id);
                }
                loadChats();
                break;
            case 'delete_message':
                const del = messages.find(m => m.id === msg.payload.message_id);
                if (del) { del.is_deleted = true; del.encrypted_content = ''; }
                renderMessages();
                break;
            case 'messages_read':
                if (currentChat && msg.payload.chat_id === currentChat.id) {
                    messages.forEach(m => { if (m.sender_id === currentUser?.id) m.status = 'read'; });
                    renderMessages();
                }
                break;
        }
    }

    // ===== УВЕДОМЛЕНИЯ =====
    function requestNotificationPermission() {
        if (!('Notification' in window)) return;
        Notification.requestPermission().then(p => notificationPermission = p);
    }
    function sendNotification(title, body) {
        if (notificationPermission !== 'granted' || isPageActive) return;
        const n = new Notification(title, { body, icon: '/web/favicon.ico' });
        n.onclick = () => { window.focus(); n.close(); };
        setTimeout(() => n.close(), 5000);
    }
    function updateUnreadBadges() {
        document.title = (totalUnread > 0 ? `(${totalUnread}) ` : '') + 'XSM — Мессенджер';
        document.querySelectorAll('.chat-item').forEach(item => {
            const chatId = item.dataset.chatId;
            const count = unreadCounts[chatId] || 0;
            let badge = item.querySelector('.unread-badge');
            if (count > 0) {
                if (!badge) { badge = document.createElement('span'); badge.className = 'unread-badge'; item.querySelector('.chat-meta').appendChild(badge); }
                badge.textContent = count > 99 ? '99+' : count;
                badge.style.display = 'flex';
            } else if (badge) badge.style.display = 'none';
        });
    }
    async function loadUnreadCounts() {
        try { const r = await apiCall('/api/v1/messages/unread'); const d = await r.json(); unreadCounts = d.unread_counts || {}; totalUnread = d.total_unread || 0; updateUnreadBadges(); } catch(e) {}
    }
    async function markChatAsRead(chatId) {
        try {
            await apiCall('/api/v1/messages/read', { method: 'POST', body: JSON.stringify({ chat_id: chatId }) });
            if (unreadCounts[chatId]) { totalUnread -= unreadCounts[chatId]; unreadCounts[chatId] = 0; updateUnreadBadges(); }
        } catch(e) {}
    }

    // ===== ПРОФИЛЬ =====
    async function loadUserInfo() {
        try {
            const r = await apiCall('/api/v1/user/profile'); const d = await r.json(); currentUser = d;
            if (navAvatar) {
                if (currentUser.avatar_url) { navAvatar.style.backgroundImage = `url(${currentUser.avatar_url})`; navAvatar.textContent = ''; }
                else { navAvatar.style.backgroundImage = ''; navAvatar.textContent = (currentUser.username || 'U')[0].toUpperCase(); }
            }
            if (document.getElementById('settings-content')?.style.display === 'block') fillSettingsForm();
        } catch(e) {}
    }

    function fillSettingsForm() {
        if (!currentUser) return;
        if (settingsUsernameDisplay) settingsUsernameDisplay.textContent = currentUser.username || '-';
        if (settingsEmailDisplay) settingsEmailDisplay.textContent = currentUser.email || '-';
        if (editUsername) editUsername.value = currentUser.username || '';
        if (editEmail) editEmail.value = currentUser.email || '';
        if (settingsStatusText) settingsStatusText.value = currentUser.status_text || '';
        if (settingsBio) settingsBio.value = currentUser.bio || '';
        if (settingsAvatarPreview) settingsAvatarPreview.src = currentUser.avatar_url || '';
    }

    // ===== ЧАТЫ =====
    async function loadChats() {
        try { const r = await apiCall('/api/v1/chats'); const d = await r.json(); chats = d.chats || []; renderChats(); } catch(e) {}
    }
    function renderChats(filter = '') {
        const filtered = chats.filter(c => (c.name || 'Чат').toLowerCase().includes(filter.toLowerCase()));
        chatsList.innerHTML = '';
        if (filtered.length === 0) {
            chatsList.innerHTML = `<div style="display:flex;flex-direction:column;align-items:center;justify-content:center;padding:2rem;text-align:center;color:var(--text-muted);"><i class="fa-solid fa-comment-slash" style="font-size:2rem;margin-bottom:0.5rem;opacity:0.4;"></i><p>${filter ? 'Ничего не найдено' : 'Нет чатов'}</p></div>`;
            return;
        }
        filtered.forEach(chat => {
            const div = document.createElement('div');
            div.className = `chat-item ${currentChat?.id === chat.id ? 'active' : ''}`;
            div.dataset.chatId = chat.id;
            const unread = unreadCounts[chat.id] || 0;
            const avatarHTML = chat.is_group 
                ? `<div class="chat-avatar">${(chat.name || 'Г')[0].toUpperCase()}</div>`
                : (chat.avatar_url 
                    ? `<div class="chat-avatar" style="background-image:url(${chat.avatar_url})"></div>` 
                    : `<div class="chat-avatar">${(chat.name || 'Ч')[0].toUpperCase()}</div>`);
            div.innerHTML = `
                ${avatarHTML}
                <div class="chat-info">
                    <div class="chat-name">${escapeHtml(chat.name || 'Чат')}</div>
                    <div class="chat-preview">${chat.is_group ? 'Группа' : chat.last_message ? escapeHtml(chat.last_message.content || '') : 'Нет сообщений'}</div>
                </div>
                <div class="chat-meta">
                    <span class="chat-time">${formatTime(chat.updated_at)}</span>
                    ${unread > 0 ? `<span class="unread-badge">${unread > 99 ? '99+' : unread}</span>` : ''}
                </div>`;
            div.addEventListener('click', () => selectChat(chat));
            div.addEventListener('contextmenu', (e) => { e.preventDefault(); showChatContextMenu(e, chat, div); });
            chatsList.appendChild(div);
        });
    }
    function selectChat(chat) {
        // Сбрасываем флаг, чтобы новый чат прокрутился вниз один раз
        if (messagesContainer) delete messagesContainer.dataset.scrolledOnce;
        currentChat = chat;
        emptyState.style.display = 'none';
        chatView.style.display = 'flex';
        partnerName.textContent = chat.name || 'Чат';
        if (chat.avatar_url) { partnerAvatar.style.backgroundImage = `url(${chat.avatar_url})`; partnerAvatar.textContent = ''; }
        else { partnerAvatar.style.backgroundImage = ''; partnerAvatar.textContent = (chat.name || 'Ч')[0].toUpperCase(); }
        partnerStatus.innerHTML = '<i class="fa-solid fa-circle" style="font-size:0.45rem;color:var(--success);"></i> В сети';
        renderChats();
        loadMessages(chat.id);
        markChatAsRead(chat.id);
        setTimeout(() => messageInput?.focus(), 100);
    }

    // ===== СООБЩЕНИЯ =====
    async function loadMessages(chatId) {
        try { const r = await apiCall(`/api/v1/messages/${chatId}`); const d = await r.json(); messages = d.messages || []; renderMessages(); } catch(e) {}
    }
    function createContextMenu() {
        if (contextMenu) contextMenu.remove();
        contextMenu = document.createElement('div');
        contextMenu.className = 'context-menu';
        contextMenu.style.display = 'none';
        contextMenu.innerHTML = `
            <div class="context-menu-item" data-action="copy"><i class="fa-solid fa-copy"></i><span>Копировать</span></div>
            <div class="context-menu-item" data-action="reply"><i class="fa-solid fa-reply"></i><span>Ответить</span></div>
            <div class="context-menu-divider"></div>
            <div class="context-menu-item danger" data-action="delete"><i class="fa-solid fa-trash"></i><span>Удалить</span></div>`;
        document.body.appendChild(contextMenu);
    }
    function renderMessages() {
        messagesList.innerHTML = '';
        if (messages.length === 0) {
            messagesList.innerHTML = `<div style="display:flex;flex-direction:column;align-items:center;justify-content:center;height:100%;text-align:center;color:var(--text-muted);padding:2rem;"><div style="width:80px;height:80px;border-radius:24px;background:var(--bg-island);border:1px solid var(--border-light);display:flex;align-items:center;justify-content:center;margin-bottom:1.5rem;"><i class="fa-solid fa-lock" style="font-size:2rem;color:var(--primary-light);opacity:0.6;"></i></div><p style="font-size:1.1rem;font-weight:600;color:var(--text-secondary);margin-bottom:0.5rem;">Нет сообщений</p><span style="font-size:0.85rem;">Чат защищён end-to-end шифрованием</span></div>`;
            return;
        }
        let lastDate = null;
        messages.forEach(msg => {
            const msgDate = new Date(msg.created_at);
            const msgDateStr = msgDate.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' });
            if (msgDateStr !== lastDate) {
                const dateDiv = document.createElement('div');
                dateDiv.className = 'messages-date-divider';
                dateDiv.innerHTML = `<span>${msgDateStr}</span>`;
                messagesList.appendChild(dateDiv);
                lastDate = msgDateStr;
            }
            const isSent = msg.sender_id === currentUser?.id;
            const isDeleted = msg.is_deleted;
            const wrapper = document.createElement('div');
            wrapper.className = `message-wrapper ${isSent ? 'sent' : 'received'}`;
            wrapper.dataset.messageId = msg.id;
            if (isDeleted) {
                wrapper.innerHTML = `<div style="padding:0.5rem 1rem;border-radius:1rem;font-size:0.85rem;color:var(--text-muted);font-style:italic;border:1px dashed var(--border-light);display:flex;align-items:center;gap:0.5rem;"><i class="fa-solid fa-ban"></i> Сообщение удалено</div><div class="message-meta"><span>${formatMessageTime(msg.created_at)}</span></div>`;
            } else {
                const bubble = document.createElement('div');
                bubble.className = 'message-bubble-wrapper';
                let replyHTML = '';
                if (msg.reply_to) {
                    const repliedMsg = messages.find(m => m.id === msg.reply_to);
                    const replyName = repliedMsg ? (repliedMsg.sender_id === currentUser?.id ? 'Вы' : (currentChat?.name || 'Собеседник')) : 'Сообщение';
                    const replyText = repliedMsg ? escapeHtml((repliedMsg.encrypted_content || '').substring(0, 50)) : '(недоступно)';
                    replyHTML = `<div class="message-reply-preview" data-reply-to="${msg.reply_to}"><div class="message-reply-name">${replyName}</div><div class="message-reply-text">${replyText}</div></div>`;
                }
                bubble.innerHTML = `${replyHTML}<div class="message-bubble"><div class="message-text">${escapeHtml(msg.encrypted_content || '')}</div></div><div class="message-meta"><span class="message-time">${formatMessageTime(msg.created_at)}</span>${isSent ? getStatusIcon(msg.status) : ''}</div>`;
                bubble.addEventListener('contextmenu', (e) => {
                    e.preventDefault(); e.stopPropagation();
                    if (!contextMenu) createContextMenu();
                    let x = e.clientX, y = e.clientY;
                    if (x + 220 > window.innerWidth) x = window.innerWidth - 230;
                    if (y + 160 > window.innerHeight) y = window.innerHeight - 170;
                    contextMenu.style.display = 'block';
                    contextMenu.style.left = x + 'px';
                    contextMenu.style.top = y + 'px';
                    contextMenu.querySelector('[data-action="copy"]').onclick = () => { navigator.clipboard.writeText(msg.encrypted_content || ''); showToast('Скопировано', 'success'); contextMenu.style.display = 'none'; };
                    contextMenu.querySelector('[data-action="reply"]').onclick = () => { replyToMessage = msg; messageInput.focus(); updateReplyPreview(); contextMenu.style.display = 'none'; };
                    const delBtn = contextMenu.querySelector('[data-action="delete"]');
                    if (isSent) { delBtn.style.display = 'flex'; delBtn.onclick = () => { contextMenu.style.display = 'none'; showDeleteConfirm(msg.id); }; }
                    else delBtn.style.display = 'none';
                    setTimeout(() => {
                        const close = (ev) => { if (!contextMenu.contains(ev.target)) { contextMenu.style.display = 'none'; document.removeEventListener('click', close); } };
                        document.addEventListener('click', close);
                    }, 0);
                });
                bubble.addEventListener('dblclick', () => { replyToMessage = msg; messageInput.focus(); updateReplyPreview(); });
                wrapper.appendChild(bubble);
            }
            messagesList.appendChild(wrapper);
        });
        // Прокрутка вниз только при первом открытии чата
        if (!messagesContainer.dataset.scrolledOnce) {
            messagesContainer.dataset.scrolledOnce = 'true';
            messagesContainer.scrollTo({ top: messagesContainer.scrollHeight, behavior: 'auto' });
        }
    }

    function getStatusIcon(status) {
        if (status === 'sent') return '<span class="message-status" style="color:var(--text-muted);"><i class="fa-solid fa-check"></i></span>';
        if (status === 'delivered') return '<span class="message-status" style="color:var(--text-muted);"><i class="fa-solid fa-check-double"></i></span>';
        if (status === 'read') return '<span class="message-status" style="color:var(--primary-light);"><i class="fa-solid fa-check-double"></i></span>';
        return '';
    }

    function showDeleteConfirm(messageId) {
        const overlay = document.createElement('div');
        overlay.className = 'delete-confirm-overlay';
        overlay.innerHTML = `<div class="delete-confirm-dialog"><p>Удалить сообщение?</p><div class="delete-confirm-actions"><button class="btn-cancel-delete">Отмена</button><button class="btn-confirm-delete">Удалить</button></div></div>`;
        document.body.appendChild(overlay);
        overlay.querySelector('.btn-cancel-delete').onclick = () => overlay.remove();
        overlay.querySelector('.btn-confirm-delete').onclick = async () => {
            overlay.remove();
            try { await apiCall(`/api/v1/messages/${messageId}`, { method: 'DELETE' }); const m = messages.find(m => m.id === messageId); if (m) { m.is_deleted = true; m.encrypted_content = ''; } renderMessages(); showToast('Удалено', 'info'); } catch(e) { console.error(e); }
        };
    }

    function updateReplyPreview() {
        const existing = document.querySelector('.reply-preview-bar');
        if (existing) existing.remove();
        if (!replyToMessage) return;
        const bar = document.createElement('div');
        bar.className = 'reply-preview-bar';
        bar.innerHTML = `<div class="reply-preview-content"><div class="reply-preview-header"><i class="fa-solid fa-reply"></i><span>Ответ на сообщение</span></div><div class="reply-preview-text">${escapeHtml((replyToMessage.encrypted_content || '').substring(0, 100))}</div></div><button class="btn-icon reply-cancel-btn"><i class="fa-solid fa-xmark"></i></button>`;
        const composer = document.querySelector('.composer');
        composer.parentNode.insertBefore(bar, composer);
        bar.querySelector('.reply-cancel-btn').onclick = () => { replyToMessage = null; bar.remove(); };
    }

    async function sendMessage() {
        const content = messageInput.value.trim();
        if (!content || !currentChat) return;
        messageInput.value = ''; autoResize();
        const tempId = 'temp-' + Date.now();
        const optimistic = { id: tempId, chat_id: currentChat.id, sender_id: currentUser?.id, encrypted_content: content, content_type: 'text', created_at: new Date().toISOString(), status: 'sent' };
        messages.push(optimistic); renderMessages();
        if (replyToMessage) { replyToMessage = null; const bar = document.querySelector('.reply-preview-bar'); if (bar) bar.remove(); }
        try {
            const res = await apiCall('/api/v1/messages', { method: 'POST', body: JSON.stringify({ chat_id: currentChat.id, encrypted_content: content, content_type: 'text' }) });
            if (res.ok) {
                const data = await res.json();
                const idx = messages.findIndex(m => m.id === tempId);
                if (idx !== -1) messages[idx] = data.message;
                await loadMessages(currentChat.id); await loadChats();
            } else {
                messages = messages.filter(m => m.id !== tempId); renderMessages();
                const err = await res.json(); showToast(err.error || 'Ошибка', 'error');
            }
        } catch(e) { messages = messages.filter(m => m.id !== tempId); renderMessages(); showToast('Ошибка сети', 'error'); }
        if (scrollBottomBtn) scrollBottomBtn.classList.remove('visible');
    }

    // ===== ПОИСК ПОЛЬЗОВАТЕЛЕЙ =====
    newChatInput?.addEventListener('input', debounce(async (e) => {
        const q = e.target.value.trim();
        if (q.length < 2) { searchResults.style.display = 'none'; createChatBtn.disabled = true; selectedUser = null; return; }
        try { const r = await apiCall(`/api/v1/users/search?q=${encodeURIComponent(q)}`); const d = await r.json(); renderSearchResults(d.users || []); } catch(ex) {}
    }, 300));

    function renderSearchResults(users) {
        searchResults.innerHTML = '';
        if (!users.length) { searchResults.style.display = 'block'; searchResults.innerHTML = '<p style="padding:1rem;text-align:center;color:var(--text-muted);">Ничего не найдено</p>'; createChatBtn.disabled = true; selectedUser = null; return; }
        searchResults.style.display = 'block';
        users.forEach(u => {
            const div = document.createElement('div');
            div.className = 'search-result-item';
            div.innerHTML = `<div class="search-result-avatar">${(u.username||'?')[0].toUpperCase()}</div><div class="search-result-info"><div class="search-result-name">${escapeHtml(u.username)}</div><div class="search-result-status">${u.status||'Не в сети'}</div></div>`;
            div.addEventListener('click', () => {
                document.querySelectorAll('.search-result-item').forEach(el => el.classList.remove('selected'));
                div.classList.add('selected'); selectedUser = u; newChatInput.value = u.username; searchResults.style.display = 'none'; createChatBtn.disabled = false;
            });
            searchResults.appendChild(div);
        });
    }

    async function createChat() {
        if (!selectedUser) return;
        try {
            const r = await apiCall('/api/v1/chats', { method: 'POST', body: JSON.stringify({ username: selectedUser.username }) });
            if (r.ok) { closeModal(); await loadChats(); }
            else { const e = await r.json(); showToast(e.error || 'Ошибка', 'error'); }
        } catch(ex) {}
    }

    function openModal() { newChatModal.classList.add('active'); setTimeout(() => newChatInput.focus(), 100); }
    function closeModal() { newChatModal.classList.remove('active'); newChatInput.value = ''; searchResults.style.display = 'none'; selectedUser = null; createChatBtn.disabled = true; }

    // ===== КОНТЕКСТНОЕ МЕНЮ ЧАТА =====
    function showChatContextMenu(e, chat, el) {
        const old = document.querySelector('.context-menu'); if (old) old.remove();
        const menu = document.createElement('div'); menu.className = 'context-menu';
        menu.innerHTML = `<div class="context-menu-item" data-action="info"><i class="fa-solid fa-info-circle"></i><span>Информация</span></div><div class="context-menu-divider"></div><div class="context-menu-item danger" data-action="delete"><i class="fa-solid fa-trash"></i><span>Удалить чат</span></div>`;
        document.body.appendChild(menu);
        let x = e.clientX, y = e.clientY;
        if (x + 220 > window.innerWidth) x = window.innerWidth - 230;
        if (y + 100 > window.innerHeight) y = window.innerHeight - 110;
        menu.style.display = 'block'; menu.style.left = x + 'px'; menu.style.top = y + 'px';
        menu.querySelector('[data-action="delete"]').onclick = async () => {
            menu.remove(); showDeleteChatConfirm(chat);
        };
        setTimeout(() => {
            const close = (ev) => { if (!menu.contains(ev.target)) { menu.remove(); document.removeEventListener('click', close); } };
            document.addEventListener('click', close);
        }, 0);
    }

    function showDeleteChatConfirm(chat) {
        const overlay = document.createElement('div'); overlay.className = 'delete-confirm-overlay';
        overlay.innerHTML = `<div class="delete-confirm-dialog"><p>Удалить чат "${escapeHtml(chat.name)}"?</p><small style="color:var(--text-muted);">Все сообщения будут удалены</small><div class="delete-confirm-actions" style="margin-top:1rem;"><button class="btn-cancel-delete">Отмена</button><button class="btn-confirm-delete">Удалить</button></div></div>`;
        document.body.appendChild(overlay);
        overlay.querySelector('.btn-cancel-delete').onclick = () => overlay.remove();
        overlay.querySelector('.btn-confirm-delete').onclick = async () => {
            overlay.remove();
            try { await apiCall(`/api/v1/chats/${chat.id}`, { method: 'DELETE' }); chats = chats.filter(c => c.id !== chat.id); if (currentChat?.id === chat.id) { currentChat = null; emptyState.style.display = 'flex'; chatView.style.display = 'none'; } renderChats(); showToast('Чат удалён', 'info'); } catch(e) {}
        };
    }

    // ===== ЭМОДЗИ =====
    function toggleEmoji() { emojiPicker.style.display = emojiPicker.style.display === 'none' ? 'block' : 'none'; }
    function insertEmoji(emoji) {
        const start = messageInput.selectionStart, end = messageInput.selectionEnd;
        messageInput.value = messageInput.value.substring(0, start) + emoji + messageInput.value.substring(end);
        messageInput.focus(); messageInput.selectionStart = messageInput.selectionEnd = start + emoji.length;
    }

    // ===== НАВИГАЦИЯ =====
    function switchPanel(panel) {
        document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
        document.getElementById(`nav-${panel}`)?.classList.add('active');
        const titleEl = document.getElementById('panel-title');
        if (titleEl) titleEl.textContent = panel === 'chats' ? 'Чаты' : panel === 'contacts' ? 'Контакты' : 'Настройки';
        const chatsListEl = document.getElementById('chats-list');
        const contactsListEl = document.getElementById('contacts-list');
        const settingsEl = document.getElementById('settings-content');
        if (chatsListEl) chatsListEl.style.display = panel === 'chats' ? 'block' : 'none';
        if (contactsListEl) contactsListEl.style.display = panel === 'contacts' ? 'block' : 'none';
        if (settingsEl) settingsEl.style.display = panel === 'settings' ? 'block' : 'none';
        const searchInputEl = document.getElementById('search-input');
        if (searchInputEl) searchInputEl.style.display = panel === 'settings' ? 'none' : 'block';
        const newChatBtnEl = document.getElementById('new-chat-btn');
        if (newChatBtnEl) newChatBtnEl.style.display = panel === 'chats' ? 'flex' : 'none';
        const searchIconEl = document.querySelector('.search-box .search-icon');
        if (searchIconEl) searchIconEl.style.display = panel === 'settings' ? 'none' : 'block';
        if (panel === 'settings') fillSettingsForm();
    }

    // ===== ПРОФИЛЬ: СОХРАНЕНИЕ =====
    saveProfileBtn?.addEventListener('click', async () => {
        const username = editUsername?.value.trim();
        const email = editEmail?.value.trim();
        const statusText = settingsStatusText?.value.trim();
        const bio = settingsBio?.value.trim();
        if (!username || !email) { showToast('Имя и email обязательны', 'error'); return; }
        try {
            const res = await apiCall('/api/v1/user/profile', { method: 'PUT', body: JSON.stringify({ username, email, status_text: statusText, bio }) });
            if (res.ok) {
                const data = await res.json(); currentUser = data.user;
                if (navAvatar) {
                    if (currentUser.avatar_url) { navAvatar.style.backgroundImage = `url(${currentUser.avatar_url})`; navAvatar.textContent = ''; }
                    else { navAvatar.style.backgroundImage = ''; navAvatar.textContent = (currentUser.username || 'U')[0].toUpperCase(); }
                }
                fillSettingsForm(); showToast('Профиль обновлён', 'success');
            } else { const err = await res.json(); showToast(err.error || 'Ошибка', 'error'); }
        } catch(e) { console.error(e); }
    });

    avatarUploadInput?.addEventListener('change', async (e) => {
        const file = e.target.files[0]; if (!file) return;
        const formData = new FormData(); formData.append('avatar', file);
        try {
            const res = await fetch('/api/v1/user/avatar', { method: 'POST', headers: { 'Authorization': `Bearer ${token}` }, body: formData });
            if (res.ok) {
                const data = await res.json();
                currentUser.avatar_url = data.avatar_url;
                if (settingsAvatarPreview) settingsAvatarPreview.src = data.avatar_url;
                if (navAvatar) { navAvatar.style.backgroundImage = `url(${data.avatar_url})`; navAvatar.textContent = ''; }
                showToast('Аватарка обновлена', 'success');
            } else showToast('Ошибка загрузки', 'error');
        } catch(e) { console.error(e); }
    });

    // ===== УТИЛИТЫ =====
    function autoResize() { const el = messageInput; el.style.height = 'auto'; el.style.height = Math.min(el.scrollHeight, 120) + 'px'; }
    function formatTime(ts) { if (!ts) return ''; const d = new Date(ts); const now = new Date(); if (now - d < 86400000 && now.getDate() === d.getDate()) return d.toLocaleTimeString('ru-RU', { hour:'2-digit', minute:'2-digit' }); return d.toLocaleDateString('ru-RU', { day:'numeric', month:'short' }); }
    function formatMessageTime(ts) { if (!ts) return ''; const d = new Date(ts); const now = new Date(); const diff = now - d; if (diff < 86400000 && now.getDate() === d.getDate()) return d.toLocaleTimeString('ru-RU', { hour:'2-digit', minute:'2-digit' }); const yesterday = new Date(now); yesterday.setDate(yesterday.getDate()-1); if (d.getDate() === yesterday.getDate() && d.getMonth() === yesterday.getMonth() && d.getFullYear() === yesterday.getFullYear()) return 'Вчера ' + d.toLocaleTimeString('ru-RU', { hour:'2-digit', minute:'2-digit' }); return d.toLocaleDateString('ru-RU', { day:'numeric', month:'short' }) + ' ' + d.toLocaleTimeString('ru-RU', { hour:'2-digit', minute:'2-digit' }); }
    function escapeHtml(text) { const map = { '&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&#039;' }; return String(text).replace(/[&<>"']/g, m => map[m]); }
    function debounce(fn, ms) { let t; return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); }; }
    function showToast(msg, type='info') {
        const old = document.querySelectorAll('.toast-notification'); if (old.length>3) old[0].remove();
        const icons = { success:'fa-circle-check', error:'fa-circle-xmark', info:'fa-circle-info', warning:'fa-triangle-exclamation' };
        const toast = document.createElement('div'); toast.className = `toast-notification toast-${type}`;
        toast.innerHTML = `<i class="fa-solid ${icons[type]||icons.info}"></i><span>${msg}</span>`;
        document.body.appendChild(toast);
        setTimeout(() => toast.classList.add('show'), 10);
        setTimeout(() => { toast.classList.remove('show'); setTimeout(() => toast.remove(), 300); }, 3000);
    }

    // ===== ОБРАБОТЧИКИ СОБЫТИЙ =====
    newChatBtn.addEventListener('click', openModal);
    createChatBtn.addEventListener('click', createChat);
    document.querySelectorAll('.modal-close-btn').forEach(b => b.addEventListener('click', closeModal));
    newChatModal.addEventListener('click', e => { if (e.target === newChatModal) closeModal(); });
    searchInput.addEventListener('input', e => renderChats(e.target.value));
    sendBtn.addEventListener('click', sendMessage);
    messageInput.addEventListener('keydown', e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage(); } });
    messageInput.addEventListener('input', autoResize);
    emojiBtn.addEventListener('click', toggleEmoji);
    emojiPicker.querySelectorAll('span').forEach(em => em.addEventListener('click', () => insertEmoji(em.textContent)));
    document.addEventListener('click', e => { if (!emojiPicker.contains(e.target) && e.target !== emojiBtn) emojiPicker.style.display = 'none'; });
    logoutBtn.addEventListener('click', () => { localStorage.removeItem('xsm_token'); window.location.href = '/web/auth.html'; });
    document.getElementById('nav-chats')?.addEventListener('click', () => switchPanel('chats'));
    document.getElementById('nav-contacts')?.addEventListener('click', () => switchPanel('contacts'));
    document.getElementById('nav-settings')?.addEventListener('click', () => switchPanel('settings'));
    document.addEventListener('visibilitychange', () => { isPageActive = !document.hidden; if (isPageActive && currentChat) markChatAsRead(currentChat.id); });
    window.addEventListener('focus', () => { isPageActive = true; if (currentChat) markChatAsRead(currentChat.id); });
    window.addEventListener('blur', () => { isPageActive = false; });

    // ===== СТАРТ =====
    requestNotificationPermission();
    loadUnreadCounts();
    setInterval(loadUnreadCounts, 30000);
    loadUserInfo();
    loadChats();
    connectWebSocket();
});
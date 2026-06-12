package websocket

import (
"encoding/json"
"log"
"sync"

"github.com/google/uuid"
)

// Message - WebSocket сообщение
type Message struct {
Type    string      `json:"type"`
Payload interface{} `json:"payload"`
}

// Client - подключённый клиент
type Client struct {
UserID uuid.UUID
Send   chan []byte
Hub    *Hub
}

// Hub - управляет подключениями
type Hub struct {
clients    map[uuid.UUID]*Client
register   chan *Client
unregister chan *Client
broadcast  chan *BroadcastMessage
mu         sync.RWMutex
}

// BroadcastMessage для отправки группе пользователей
type BroadcastMessage struct {
UserIDs []uuid.UUID
Message Message
}

func NewHub() *Hub {
return &Hub{
clients:    make(map[uuid.UUID]*Client),
register:   make(chan *Client),
unregister: make(chan *Client),
broadcast:  make(chan *BroadcastMessage, 256),
}
}

func (h *Hub) Run() {
for {
select {
case client := <-h.register:
h.mu.Lock()
h.clients[client.UserID] = client
count := len(h.clients)
h.mu.Unlock()
log.Printf("👤 Пользователь подключился (онлайн: %d)", count)

case client := <-h.unregister:
h.mu.Lock()
if _, ok := h.clients[client.UserID]; ok {
delete(h.clients, client.UserID)
close(client.Send)
}
count := len(h.clients)
h.mu.Unlock()
log.Printf("👋 Пользователь отключился (онлайн: %d)", count)

case msg := <-h.broadcast:
h.sendToUsers(msg.UserIDs, msg.Message)
}
}
}

func (h *Hub) SendToUser(userID uuid.UUID, msg Message) {
h.mu.RLock()
client, ok := h.clients[userID]
h.mu.RUnlock()
if !ok {
return
}

data, err := json.Marshal(msg)
if err != nil {
return
}

select {
case client.Send <- data:
default:
h.mu.Lock()
delete(h.clients, userID)
close(client.Send)
h.mu.Unlock()
}
}

func (h *Hub) SendToUsers(userIDs []uuid.UUID, msg Message) {
for _, userID := range userIDs {
h.SendToUser(userID, msg)
}
}

func (h *Hub) sendToUsers(userIDs []uuid.UUID, msg Message) {
h.SendToUsers(userIDs, msg)
}

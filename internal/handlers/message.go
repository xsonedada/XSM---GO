package handlers

import (
	"database/sql"
	"net/http"
	"time"

	ws "xsm/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type MessageHandler struct {
	db  *sql.DB
	hub *ws.Hub
}

func NewMessageHandler(db *sql.DB, hub *ws.Hub) *MessageHandler {
	return &MessageHandler{db: db, hub: hub}
}

// SendMessage отправляет сообщение
func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	senderID := userID.(uuid.UUID)

	var req struct {
		ChatID           string `json:"chat_id" binding:"required"`
		EncryptedContent string `json:"encrypted_content" binding:"required"`
		ContentType      string `json:"content_type"`
		IV               []byte `json:"iv"`
		ReplyTo          string `json:"reply_to"` // ID сообщения, на которое отвечаем
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат данных"})
		return
	}

	chatID, err := uuid.Parse(req.ChatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID чата"})
		return
	}

	// Проверяем участие в чате
	var isParticipant bool
	err = h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id = $1 AND user_id = $2)
	`, chatID, senderID).Scan(&isParticipant)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "вы не участник этого чата"})
		return
	}

	if req.ContentType == "" {
		req.ContentType = "text"
	}
	if req.IV == nil {
		req.IV = make([]byte, 16)
	}

	// Парсим reply_to, если указан
	var replyToUUID *uuid.UUID
	if req.ReplyTo != "" {
		if parsed, err := uuid.Parse(req.ReplyTo); err == nil {
			replyToUUID = &parsed
		}
	}

	// Сохраняем сообщение
	var messageID uuid.UUID
	var createdAt time.Time
	err = h.db.QueryRow(`
		INSERT INTO messages (chat_id, sender_id, encrypted_content, content_type, iv, reply_to, read_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, chatID, senderID, req.EncryptedContent, req.ContentType, req.IV, replyToUUID, pq.Array([]uuid.UUID{senderID})).
		Scan(&messageID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось отправить сообщение"})
		return
	}

	// Обновляем время чата
	h.db.Exec("UPDATE chats SET updated_at = $1 WHERE id = $2", createdAt, chatID)

	// Получаем участников чата
	participants := h.getChatParticipants(chatID, senderID)

	// Отправляем уведомление через WebSocket
	wsMsg := ws.Message{
		Type: "new_message",
		Payload: gin.H{
			"id":                messageID,
			"chat_id":           chatID,
			"sender_id":         senderID,
			"encrypted_content": req.EncryptedContent,
			"content_type":      req.ContentType,
			"iv":                req.IV,
			"created_at":        createdAt,
			"status":            "sent",
			"read_by":           []uuid.UUID{senderID},
			"reply_to":          req.ReplyTo, // передаём клиенту
		},
	}
	h.hub.SendToUsers(participants, wsMsg)

	c.JSON(http.StatusCreated, gin.H{
		"message": gin.H{
			"id":                messageID,
			"chat_id":           chatID,
			"sender_id":         senderID,
			"encrypted_content": req.EncryptedContent,
			"content_type":      req.ContentType,
			"iv":                req.IV,
			"created_at":        createdAt,
			"status":            "sent",
			"reply_to":          req.ReplyTo,
		},
	})
}

// MarkAsRead отмечает сообщения как прочитанные
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUserID := userID.(uuid.UUID)

	var req struct {
		ChatID string `json:"chat_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный формат"})
		return
	}

	chatID, err := uuid.Parse(req.ChatID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID чата"})
		return
	}

	// Отмечаем все непрочитанные сообщения как прочитанные
	_, err = h.db.Exec(`
		UPDATE messages 
		SET read_by = array_append(read_by, $1)
		WHERE chat_id = $2 
		AND sender_id != $1 
		AND NOT ($1 = ANY(read_by))
	`, currentUserID, chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка"})
		return
	}

	// Уведомляем отправителей, что сообщения прочитаны
	participants := h.getChatParticipants(chatID, currentUserID)
	wsMsg := ws.Message{
		Type: "messages_read",
		Payload: gin.H{
			"chat_id": chatID,
			"user_id": currentUserID,
		},
	}
	h.hub.SendToUsers(participants, wsMsg)

	c.JSON(http.StatusOK, gin.H{"message": "отмечено как прочитанное"})
}

// GetMessages возвращает сообщения чата
func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUserID := userID.(uuid.UUID)

	chatIDStr := c.Param("chat_id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID чата"})
		return
	}

	var isParticipant bool
	err = h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM chat_participants WHERE chat_id = $1 AND user_id = $2)
	`, chatID, currentUserID).Scan(&isParticipant)
	if err != nil || !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "доступ запрещён"})
		return
	}

	rows, err := h.db.Query(`
		SELECT id, chat_id, sender_id, encrypted_content, content_type, iv, 
		       is_edited, is_deleted, created_at, updated_at, read_by, reply_to
		FROM messages
		WHERE chat_id = $1
		ORDER BY created_at ASC
		LIMIT 100
	`, chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка загрузки сообщений"})
		return
	}
	defer rows.Close()

	var messages []gin.H
	for rows.Next() {
		var (
			id, msgChatID, senderID       uuid.UUID
			encryptedContent, contentType string
			iv                            []byte
			isEdited, isDeleted           bool
			createdAt, updatedAt          time.Time
			readBy                        []string
			replyTo                       uuid.NullUUID
		)

		if err := rows.Scan(&id, &msgChatID, &senderID, &encryptedContent, &contentType,
			&iv, &isEdited, &isDeleted, &createdAt, &updatedAt, pq.Array(&readBy), &replyTo); err != nil {
			continue
		}

		// Определяем статус сообщения
		status := "sent"
		isRead := false
		for _, readerID := range readBy {
			uid, _ := uuid.Parse(readerID)
			if uid != senderID {
				isRead = true
				break
			}
		}

		if isRead {
			status = "read"
		} else {
			// Если есть другие участники, считаем доставленным (клиент сам не знает)
			status = "delivered"
		}

		replyToStr := ""
		if replyTo.Valid {
			replyToStr = replyTo.UUID.String()
		}

		messages = append(messages, gin.H{
			"id":                id,
			"chat_id":           msgChatID,
			"sender_id":         senderID,
			"encrypted_content": encryptedContent,
			"content_type":      contentType,
			"iv":                iv,
			"is_edited":         isEdited,
			"is_deleted":        isDeleted,
			"created_at":        createdAt,
			"updated_at":        updatedAt,
			"status":            status,
			"read_by":           readBy,
			"reply_to":          replyToStr,
		})
	}

	if messages == nil {
		messages = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// DeleteMessage удаляет сообщение
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUserID := userID.(uuid.UUID)

	messageIDStr := c.Param("id")
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID сообщения"})
		return
	}

	var senderID uuid.UUID
	var chatID uuid.UUID
	err = h.db.QueryRow("SELECT sender_id, chat_id FROM messages WHERE id = $1", messageID).
		Scan(&senderID, &chatID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "сообщение не найдено"})
		return
	}

	if senderID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "можно удалять только свои сообщения"})
		return
	}

	_, err = h.db.Exec("UPDATE messages SET is_deleted = true, updated_at = $1 WHERE id = $2", time.Now(), messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось удалить сообщение"})
		return
	}

	participants := h.getChatParticipants(chatID, currentUserID)
	wsMsg := ws.Message{
		Type: "delete_message",
		Payload: gin.H{
			"message_id": messageID,
			"chat_id":    chatID,
		},
	}
	h.hub.SendToUsers(participants, wsMsg)

	c.JSON(http.StatusOK, gin.H{"message": "сообщение удалено"})
}

// GetUnreadCount возвращает количество непрочитанных сообщений
func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUserID := userID.(uuid.UUID)

	rows, err := h.db.Query(`
		SELECT c.id, COALESCE(COUNT(m.id), 0) as unread
		FROM chats c
		JOIN chat_participants cp ON c.id = cp.chat_id AND cp.user_id = $1
		LEFT JOIN messages m ON c.id = m.chat_id 
			AND m.sender_id != $1 
			AND NOT (COALESCE($1 = ANY(m.read_by), false))
			AND m.is_deleted = false
		GROUP BY c.id
	`, currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	unreadCounts := make(map[string]int)
	totalUnread := 0

	for rows.Next() {
		var chatID uuid.UUID
		var count int
		if err := rows.Scan(&chatID, &count); err != nil {
			continue
		}
		unreadCounts[chatID.String()] = count
		totalUnread += count
	}

	c.JSON(http.StatusOK, gin.H{
		"unread_counts": unreadCounts,
		"total_unread":  totalUnread,
	})
}

func (h *MessageHandler) getChatParticipants(chatID uuid.UUID, excludeUserID uuid.UUID) []uuid.UUID {
	rows, err := h.db.Query(`
		SELECT user_id FROM chat_participants 
		WHERE chat_id = $1 AND user_id != $2
	`, chatID, excludeUserID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var participants []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err == nil {
			participants = append(participants, userID)
		}
	}
	return participants
}

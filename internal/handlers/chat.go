package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	db *sql.DB
}

func NewChatHandler(db *sql.DB) *ChatHandler {
	return &ChatHandler{db: db}
}

// CreateChat создаёт новый чат
func (h *ChatHandler) CreateChat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUserID := userID.(uuid.UUID)

	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "укажите имя пользователя"})
		return
	}

	// Ищем пользователя по username
	var partnerID uuid.UUID
	var partnerUsername string
	err := h.db.QueryRow("SELECT id, username FROM users WHERE username = $1", req.Username).
		Scan(&partnerID, &partnerUsername)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "пользователь не найден"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка сервера"})
		return
	}

	// Проверяем, нет ли уже чата между этими пользователями
	var existingChatID uuid.UUID
	err = h.db.QueryRow(`
		SELECT c.id FROM chats c
		JOIN chat_participants cp1 ON c.id = cp1.chat_id AND cp1.user_id = $1
		JOIN chat_participants cp2 ON c.id = cp2.chat_id AND cp2.user_id = $2
		WHERE c.is_group = false
	`, currentUserID, partnerID).Scan(&existingChatID)

	if err == nil {
		// Чат уже существует, возвращаем его
		c.JSON(http.StatusOK, gin.H{
			"chat": gin.H{
				"id":   existingChatID,
				"name": partnerUsername,
			},
			"message": "чат уже существует",
		})
		return
	}

	// Создаём новый чат
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка сервера"})
		return
	}
	defer tx.Rollback()

	var chatID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO chats (name, is_group, created_by)
		VALUES ($1, false, $2)
		RETURNING id
	`, partnerUsername, currentUserID).Scan(&chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось создать чат"})
		return
	}

	// Добавляем участников
	_, err = tx.Exec("INSERT INTO chat_participants (chat_id, user_id, role) VALUES ($1, $2, 'member')", chatID, currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка сервера"})
		return
	}
	_, err = tx.Exec("INSERT INTO chat_participants (chat_id, user_id, role) VALUES ($1, $2, 'member')", chatID, partnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка сервера"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка сервера"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"chat": gin.H{
			"id":         chatID,
			"name":       partnerUsername,
			"is_group":   false,
			"created_by": currentUserID,
		},
		"message": "чат создан",
	})
}

// GetChats возвращает список чатов пользователя с правильными названиями
func (h *ChatHandler) GetChats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUserID := userID.(uuid.UUID)

	rows, err := h.db.Query(`
		SELECT 
			c.id, 
			c.is_group, 
			c.created_at, 
			c.updated_at
		FROM chats c
		JOIN chat_participants cp ON c.id = cp.chat_id
		WHERE cp.user_id = $1
		ORDER BY c.updated_at DESC
	`, currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка загрузки чатов"})
		return
	}
	defer rows.Close()

	var chats []gin.H
	for rows.Next() {
		var chatID uuid.UUID
		var isGroup bool
		var createdAt, updatedAt sql.NullTime

		if err := rows.Scan(&chatID, &isGroup, &createdAt, &updatedAt); err != nil {
			continue
		}

		// Определяем название чата
		chatName := "Чат"

		if isGroup {
			// Для групповых чатов используем сохранённое название
			h.db.QueryRow("SELECT name FROM chats WHERE id = $1", chatID).Scan(&chatName)
		} else {
			// Для личных чатов показываем имя собеседника
			var partnerUsername string
			err := h.db.QueryRow(`
				SELECT u.username 
				FROM users u
				JOIN chat_participants cp ON u.id = cp.user_id
				WHERE cp.chat_id = $1 AND cp.user_id != $2
			`, chatID, currentUserID).Scan(&partnerUsername)
			var avatarURL sql.NullString
			if !isGroup {
				err = h.db.QueryRow(`
        SELECT u.avatar_url FROM users u
        JOIN chat_participants cp ON u.id = cp.user_id
        WHERE cp.chat_id = $1 AND cp.user_id != $2
    `, chatID, currentUserID).Scan(&avatarURL)
			}
			if err == nil {
				chatName = partnerUsername
			}
		}

		// Получаем последнее сообщение
		var lastMessage gin.H
		var msgContent sql.NullString
		var msgTime sql.NullTime

		err = h.db.QueryRow(`
			SELECT encrypted_content, created_at
			FROM messages
			WHERE chat_id = $1 AND is_deleted = false
			ORDER BY created_at DESC
			LIMIT 1
		`, chatID).Scan(&msgContent, &msgTime)

		if err == nil && msgContent.Valid {
			preview := msgContent.String
			if len(preview) > 50 {
				preview = preview[:47] + "..."
			}
			lastMessage = gin.H{
				"content":    preview,
				"created_at": msgTime.Time,
			}
		}

		chats = append(chats, gin.H{
			"id":           chatID,
			"name":         chatName,
			"is_group":     isGroup,
			"created_at":   createdAt.Time,
			"updated_at":   updatedAt.Time,
			"last_message": lastMessage,
		})
	}

	if chats == nil {
		chats = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"chats": chats})
}

// DeleteChat удаляет чат (только для создателя)
func (h *ChatHandler) DeleteChat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	currentUserID := userID.(uuid.UUID)

	chatIDStr := c.Param("id")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "неверный ID чата"})
		return
	}

	// Проверяем, что пользователь — создатель чата
	var createdBy uuid.UUID
	err = h.db.QueryRow("SELECT created_by FROM chats WHERE id = $1", chatID).Scan(&createdBy)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "чат не найден"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка сервера"})
		return
	}

	if createdBy != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "только создатель может удалить чат"})
		return
	}

	// Удаляем чат (каскадно удалятся участники и сообщения)
	_, err = h.db.Exec("DELETE FROM chats WHERE id = $1", chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "не удалось удалить чат"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "чат удалён"})
}

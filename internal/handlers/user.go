package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	db *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{db: db}
}

// SearchUsers ищет пользователей по username
func (h *UserHandler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	if len(query) < 2 {
		c.JSON(http.StatusOK, gin.H{"users": []gin.H{}})
		return
	}

	currentUserID, _ := c.Get("user_id")

	rows, err := h.db.Query(`
		SELECT id, username, status, last_seen
		FROM users
		WHERE username ILIKE $1 AND id != $2
		ORDER BY username
		LIMIT 20
	`, "%"+query+"%", currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ошибка поиска"})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var id uuid.UUID
		var username, status string
		var lastSeen sql.NullTime

		if err := rows.Scan(&id, &username, &status, &lastSeen); err != nil {
			continue
		}

		lastSeenStr := ""
		if lastSeen.Valid {
			lastSeenStr = lastSeen.Time.Format("2006-01-02 15:04")
		}

		users = append(users, gin.H{
			"id":        id,
			"username":  username,
			"status":    status,
			"last_seen": lastSeenStr,
		})
	}

	if users == nil {
		users = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

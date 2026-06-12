package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                  uuid.UUID      `json:"id" db:"id"`
	Username            string         `json:"username" db:"username"`
	Email               string         `json:"email" db:"email"`
	PasswordHash        string         `json:"-" db:"password_hash"`
	PublicKey           string         `json:"public_key" db:"public_key"`
	PrivateKeyEncrypted string         `json:"-" db:"private_key_encrypted"`
	AvatarURL           sql.NullString `json:"-" db:"avatar_url"`
	Status              string         `json:"status" db:"status"`
	LastSeen            time.Time      `json:"last_seen" db:"last_seen"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	PublicKey string    `json:"public_key"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse преобразует User в UserResponse (безопасный для API)
func (u *User) ToResponse() UserResponse {
	avatarURL := ""
	if u.AvatarURL.Valid {
		avatarURL = u.AvatarURL.String
	}
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		PublicKey: u.PublicKey,
		AvatarURL: avatarURL,
		Status:    u.Status,
		LastSeen:  u.LastSeen,
		CreatedAt: u.CreatedAt,
	}
}

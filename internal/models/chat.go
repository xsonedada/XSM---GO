package models

import (
    "time"
    "github.com/google/uuid"
)

type Chat struct {
    ID        uuid.UUID `json:"id"`
    Name      string    `json:"name,omitempty"`
    IsGroup   bool      `json:"is_group"`
    CreatedBy uuid.UUID `json:"created_by"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Participants []User `json:"participants,omitempty"`
    LastMessage *Message `json:"last_message,omitempty"`
}

type CreateChatRequest struct {
    UserID     string   `json:"user_id" binding:"required"`
    Name       string   `json:"name,omitempty"`
    IsGroup    bool     `json:"is_group"`
    Participants []string `json:"participants" binding:"required"`
}

type WebSocketMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

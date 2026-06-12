package models

import (
    "time"
    "github.com/google/uuid"
)

type Message struct {
    ID               uuid.UUID  `json:"id"`
    ChatID           uuid.UUID  `json:"chat_id"`
    SenderID         uuid.UUID  `json:"sender_id"`
    EncryptedContent string     `json:"encrypted_content"`
    ContentType      string     `json:"content_type"`
    IV               []byte     `json:"iv"`
    Signature        string     `json:"signature,omitempty"`
    ReplyTo          *uuid.UUID `json:"reply_to,omitempty"`
    IsEdited         bool       `json:"is_edited"`
    IsDeleted        bool       `json:"is_deleted"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}

type MessageRequest struct {
    ChatID           string  `json:"chat_id" binding:"required"`
    EncryptedContent string  `json:"encrypted_content" binding:"required"`
    ContentType      string  `json:"content_type"`
    IV               []byte  `json:"iv" binding:"required"`
    Signature        string  `json:"signature,omitempty"`
    ReplyTo          string  `json:"reply_to,omitempty"`
}

type MessageStatus struct {
    MessageID   uuid.UUID `json:"message_id"`
    UserID      uuid.UUID `json:"user_id"`
    Status      string    `json:"status"`
    DeliveredAt *time.Time `json:"delivered_at,omitempty"`
    ReadAt      *time.Time `json:"read_at,omitempty"`
}

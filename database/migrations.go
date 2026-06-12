package database

import (
	"log"
)

func RunMigrations(db *Database) error {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,

		`CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            username VARCHAR(50) UNIQUE NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            public_key TEXT NOT NULL,
            private_key_encrypted TEXT NOT NULL,
            avatar_url VARCHAR(500),
            status VARCHAR(20) DEFAULT 'offline',
            last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );`,

		`CREATE TABLE IF NOT EXISTS chats (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            name VARCHAR(100),
            is_group BOOLEAN DEFAULT FALSE,
            created_by UUID REFERENCES users(id),
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );`,

		`CREATE TABLE IF NOT EXISTS chat_participants (
            chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
            user_id UUID REFERENCES users(id) ON DELETE CASCADE,
            role VARCHAR(20) DEFAULT 'member',
            joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (chat_id, user_id)
        );`,

		`CREATE TABLE IF NOT EXISTS messages (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
            sender_id UUID REFERENCES users(id),
            encrypted_content TEXT NOT NULL,
            content_type VARCHAR(20) DEFAULT 'text',
            iv BYTEA NOT NULL,
            signature TEXT,
            reply_to UUID REFERENCES messages(id),
            is_edited BOOLEAN DEFAULT FALSE,
            is_deleted BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            read_by UUID[] DEFAULT '{}'
        );`,

		`CREATE TABLE IF NOT EXISTS message_status (
            message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
            user_id UUID REFERENCES users(id) ON DELETE CASCADE,
            status VARCHAR(20) DEFAULT 'sent',
            delivered_at TIMESTAMP WITH TIME ZONE,
            read_at TIMESTAMP WITH TIME ZONE,
            PRIMARY KEY (message_id, user_id)
        );`,

		`CREATE TABLE IF NOT EXISTS refresh_tokens (
            id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
            user_id UUID REFERENCES users(id) ON DELETE CASCADE,
            token VARCHAR(500) UNIQUE NOT NULL,
            expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );`,

		`-- Таблица каналов
CREATE TABLE IF NOT EXISTS channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_by UUID REFERENCES users(id),
    is_private BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);`,

		`-- Подписчики каналов
CREATE TABLE IF NOT EXISTS channel_subscribers (
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'subscriber',
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (channel_id, user_id)
);`,

		`-- Сообщения каналов
CREATE TABLE IF NOT EXISTS channel_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    sender_id UUID REFERENCES users(id),
    encrypted_content TEXT NOT NULL,
    content_type VARCHAR(20) DEFAULT 'text',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);`,

		`-- Таблица для отслеживания прочтения
CREATE TABLE IF NOT EXISTS message_reads (
    message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, user_id)
);`,

		`CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_chat_participants_user_id ON chat_participants(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);`,
	}

	for _, migration := range migrations {
		if _, err := db.DB.Exec(migration); err != nil {
			return err
		}
	}

	log.Println("✅ Database migrations completed successfully")
	return nil
}

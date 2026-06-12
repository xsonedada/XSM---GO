package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"xsm/internal/config"
)

type Database struct {
	DB *sql.DB
}

func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("✅ Database connected successfully")

	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}

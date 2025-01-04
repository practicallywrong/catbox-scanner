package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type Database struct {
	conn *sql.DB
}

func NewDatabase(connStr string) (*Database, error) {
	// Open PostgreSQL connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Set max connections if needed
	db.SetMaxOpenConns(1)

	// Ensure the valid_ids table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS valid_ids (
			id TEXT PRIMARY KEY,
			url TEXT,
			ext TEXT
		)
	`)
	if err != nil {
		return nil, err
	}

	return &Database{conn: db}, nil
}

func (db *Database) SaveValidLink(id, ext string) error {
	url := fmt.Sprintf("https://files.catbox.moe/%s%s", id, ext)

	// PostgreSQL INSERT statement
	_, err := db.conn.Exec("INSERT INTO valid_ids (id, url, ext) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING", id, url, ext)
	return err
}

func (db *Database) Close() {
	err := db.conn.Close()
	if err != nil {
		log.Printf("Error closing database connection: %v", err)
	}
}

package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	conn     *sql.DB
	saveStmt *sql.Stmt
}

// NewDatabase initializes a new SQLite database connection.
func NewDatabase(connStr string) (*Database, error) {
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS 
	found_ids 
	(
		id CHAR(6) NOT NULL UNIQUE,
		ext VARCHAR(10) NOT NULL
	);
	`
	if _, err := db.Exec(createTableQuery); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	saveStmt, err := db.Prepare("INSERT OR IGNORE INTO found_ids (id, ext) VALUES (?, ?);")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare save statement: %w", err)
	}

	return &Database{conn: db, saveStmt: saveStmt}, nil
}

func (db *Database) SaveValidLink(id, ext string) error {
	if _, err := db.saveStmt.Exec(id, ext); err != nil {
		return fmt.Errorf("failed to execute save statement: %w", err)
	}
	return nil
}

func (db *Database) Close() {
	if db.saveStmt != nil {
		if err := db.saveStmt.Close(); err != nil {
			log.Printf("Error closing save statement: %v", err)
		}
	}
	if err := db.conn.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}
}

package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	conn *sql.DB
}

func NewDatabase(filepath string) (*Database, error) {
	db, err := sql.Open("sqlite3", filepath+"?cache=shared")
	db.SetMaxOpenConns(1)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS valid_ids (
		id TEXT PRIMARY KEY,
		url TEXT,
		ext TEXT
	)`)
	if err != nil {
		return nil, err
	}
	return &Database{conn: db}, nil
}

func (db *Database) SaveValidLink(id, ext string) error {
	url := fmt.Sprintf("https://files.catbox.moe/%s%s", id, ext)
	_, err := db.conn.Exec("INSERT OR IGNORE INTO valid_ids (id, url, ext) VALUES (?, ?, ?)", id, url, ext)
	return err
}

func (db *Database) Close() {
	db.conn.Close()
}

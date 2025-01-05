package database

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type Database struct {
	conn *sql.DB
}

func NewDatabase(connStr string) (*Database, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS found_ids (
			row_number SERIAL PRIMARY KEY,
			id CHAR(6) NOT NULL UNIQUE,
			ext VARCHAR(10) NOT NULL
		)
	`)
	if err != nil {
		return nil, err
	}

	return &Database{conn: db}, nil
}

func (db *Database) SaveValidLink(id, ext string) error {
	_, err := db.conn.Exec("INSERT INTO found_ids (id, ext) VALUES ($1, $2)", id, ext)
	return err
}

func (db *Database) Close() {
	err := db.conn.Close()
	if err != nil {
		log.Printf("Error closing database connection: %v", err)
	}
}

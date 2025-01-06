package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Database wraps the GORM DB connection.
type Database struct {
	conn *gorm.DB
}

// FoundID represents the table schema.
type FoundID struct {
	ID  string `gorm:"type:char(6);unique;not null"`
	Ext string `gorm:"type:varchar(10);not null"`
}

// NewDatabase initializes a new database connection.
func NewDatabase(dialect, connStr string) (*Database, error) {
	var dialector gorm.Dialector

	switch dialect {
	case "postgres":
		dialector = postgres.Open(connStr)
	case "sqlite":
		dialector = sqlite.Open(connStr)
	default:
		return nil, fmt.Errorf("unsupported database dialect: %s", dialect)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema.
	if err := db.AutoMigrate(&FoundID{}); err != nil {
		return nil, err
	}

	return &Database{conn: db}, nil
}

// SaveValidLink adds a new record to the found_ids table.
func (db *Database) SaveValidLink(id, ext string) error {
	return db.conn.Create(&FoundID{ID: id, Ext: ext}).Error
}

// Close closes the database connection.
func (db *Database) Close() {
	sqlDB, err := db.conn.DB()
	if err != nil {
		log.Printf("Error getting raw DB: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}
}

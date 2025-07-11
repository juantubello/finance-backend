package config

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// First we create a MAP (to store [KEY][VALUE] -> Key is the name of the database and *gorm.DB is a reference from an already open database conection)
var DBs map[string]*gorm.DB = make(map[string]*gorm.DB)

func ConnectDB(alias string, path string) (*gorm.DB, error) {

	// Singleton - Check if DB connection already exists
	if db, exists := DBs[alias]; exists {
		return db, nil
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})

	if err != nil {
		return nil, fmt.Errorf("error trying to connect to database: %w", err)
	}

	DBs[alias] = db
	return db, nil
}

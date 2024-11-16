package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"sync"
)

var DB *sql.DB
var DBMutex sync.Mutex

// connectDB establishes a connection to the SQLite database
func Connect() {
	var err error
	DB, err = sql.Open("sqlite3", "./pes-people-2024-11-01.db")
	if err != nil {
		log.Fatal(err)
	}
}

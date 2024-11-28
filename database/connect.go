package database

import (
	"database/sql"
	"log"
	"sync"

	"github.com/anuragrao04/qr-attendance-backend/models"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *sql.DB
var DBMutex sync.Mutex

var GORMDB *gorm.DB
var GORMDBMutex sync.Mutex

// connectDB establishes a connection to the SQLite database
func Connect() {
	var err error
	DB, err = sql.Open("sqlite3", "./pes-people.db")
	if err != nil {
		log.Fatal(err)
	}
}

func ConnectGORM() {
	var err error
	GORMDB, err = gorm.Open(sqlite.Open("sessions.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to sessions database")
	}
	GORMDB.AutoMigrate(&models.UserFingerprint{})
}

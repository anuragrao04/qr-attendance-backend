package sessions

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/anuragrao04/qr-attendance-backend/database"
	"github.com/anuragrao04/qr-attendance-backend/models"
)

var Sessions = make(map[uint32]models.Session) // SessionID -> Session
var SessionsMutex sync.Mutex

// generates a new session of the given classroom, populating the student details on the way
func CreateSession(classroomTableName string) (uint32, error) {
	students, err := database.GetStudentsInAClassroom(classroomTableName)
	if err != nil {
		log.Println("Failed to get students in classroom:", err)
		return 0, err
	}
	// create a unique sessionID
	sessID := uint32(rand.Uint32())
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()
	Sessions[sessID] = models.Session{
		ClassroomTable: classroomTableName,
		Students:       students,
	}
	log.Println("Created new session with ID:", sessID)
	return sessID, nil
}

// updates the current random ID for a session and archives the previous one
func UpdateRandomID(sessionID uint32, newRandomID models.RandomID) error {
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()

	// Retrieve the session by ID
	session, exists := Sessions[sessionID]
	if !exists {
		return errors.New("session not found")
	}

	// Mark the current random ID as expired and archive it
	if session.CurrentRandomID.ID != 0 {
		session.CurrentRandomID.ExpiredAt = time.Now().UnixMilli() // Set expiration timestamp
		session.PastRandomIDs = append(session.PastRandomIDs, session.CurrentRandomID)
	}

	// Update the session with the new random ID
	session.CurrentRandomID = newRandomID

	// Save the updated session back to the map
	Sessions[sessionID] = session
	return nil
}

func DeleteSession(sessionID uint32) {
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()
	delete(Sessions, sessionID)
}

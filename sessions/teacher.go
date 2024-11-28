package sessions

import (
	"errors"
	"fmt"
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
func CreateSession(classroomTableName string, teacherQRRenderingLatency int64) (uint32, []models.StudentInASession, error) {
	students, err := database.GetStudentsInAClassroom(classroomTableName)
	if err != nil {
		return 0, nil, err
	}

	log.Println("Teacher Rendering Latency: ", teacherQRRenderingLatency)

	// create a unique sessionID
	sessID := uint32(rand.Uint32())
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()
	Sessions[sessID] = models.Session{
		ClassroomTable:            classroomTableName,
		Students:                  students,
		TeacherQRRenderingLatency: teacherQRRenderingLatency,
	}
	log.Println("Created new session with ID:", sessID)
	return sessID, students, nil
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

// returns absentee list and presentee list
func GetAttendanceList(sessionID uint32) ([]models.StudentInASession, []models.StudentInASession, error) {
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()

	session, exists := Sessions[sessionID]
	if !exists {
		return nil, nil, fmt.Errorf("session %d not found", sessionID)
	}

	if len(session.Students) == 0 {
		// this classroom doesn't exist
		return nil, nil, fmt.Errorf("classroom doesn't exist")
	}

	var absentees []models.StudentInASession
	var presentees []models.StudentInASession
	for _, student := range session.Students {
		if !student.IsPresent {
			absentees = append(absentees, student)
		} else {
			presentees = append(presentees, student)
		}
	}

	return absentees, presentees, nil
}

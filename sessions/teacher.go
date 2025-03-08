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
		log.Println("Failed to get students in classroom:", err)
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
	go notifyAttendanceChange(sessID) // push the first attendance list
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

func ToggleStudentAttendance(sessionID uint32, srn string) error {
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()

	session, exists := Sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %d not found", sessionID)
	}

	// Find and toggle the student's presence
	found := false
	for i, student := range session.Students {
		if student.SRN == srn {
			session.Students[i].IsPresent = !session.Students[i].IsPresent
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("student SRN %s not found in session %d", srn, sessionID)
	}

	// Save back the updated session
	Sessions[sessionID] = session

	// Notify about the change in a separate goroutine
	go notifyAttendanceChange(sessionID)

	return nil
}

// AttendanceChangeEvent represents a change in the attendance status
type AttendanceChangeEvent struct {
	SessionID  uint32
	Absentees  []models.StudentInASession
	Presentees []models.StudentInASession
}

// We'll use a map to store channels for each session
var (
	sessionEventChannels = make(map[uint32]chan AttendanceChangeEvent)
	eventChannelsMutex   sync.Mutex
)

// RegisterForAttendanceChanges creates and returns a channel that will receive
// attendance change events for the specified session
func RegisterForAttendanceChanges(sessionID uint32) chan AttendanceChangeEvent {
	eventChannelsMutex.Lock()
	defer eventChannelsMutex.Unlock()

	// Create a buffered channel to prevent blocking
	ch := make(chan AttendanceChangeEvent, 10)
	sessionEventChannels[sessionID] = ch
	return ch
}

// UnregisterFromAttendanceChanges removes the event channel for a session
func UnregisterFromAttendanceChanges(sessionID uint32) {
	eventChannelsMutex.Lock()
	defer eventChannelsMutex.Unlock()

	if ch, exists := sessionEventChannels[sessionID]; exists {
		close(ch)
		delete(sessionEventChannels, sessionID)
	}
}

// notifyAttendanceChange sends an attendance change event to any registered listeners
func notifyAttendanceChange(sessionID uint32) {
	eventChannelsMutex.Lock()
	ch, exists := sessionEventChannels[sessionID]
	eventChannelsMutex.Unlock()

	if !exists {
		return
	}

	// Get current attendance lists
	absentees, presentees, err := GetAttendanceList(sessionID)
	if err != nil {
		log.Printf("Failed to get attendance lists for notification: %v", err)
		return
	}

	// Send event to channel in a non-blocking way
	select {
	case ch <- AttendanceChangeEvent{
		SessionID:  sessionID,
		Absentees:  absentees,
		Presentees: presentees,
	}:
		// Event sent successfully
	default:
		// Channel buffer is full, log and continue
		log.Printf("Event channel for session %d is full, dropping notification", sessionID)
	}
}

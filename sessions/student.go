package sessions

import (
	"errors"
	"fmt"
	"log"
	"slices"
	"strconv"

	"github.com/anuragrao04/qr-attendance-backend/models"
)

func ValidateScan(scan models.ScanMessage, clockDrift int64, studentLatency int64) (bool, error) {
	// Fetch the session
	SessionsMutex.Lock()
	session, exists := Sessions[scan.SessionID]
	SessionsMutex.Unlock()

	if !exists {
		return false, errors.New("Invalid session ID")
	}

	// Check if the student is already marked present
	for _, student := range session.Students {
		if student.SRN == scan.SRN && student.IsPresent {
			return false, errors.New("Student already marked present")
		}
	}

	// Adjust ScannedAt for both clock drift and teacher clock drift
	int64ScannedAt, _ := strconv.ParseInt(scan.ScannedAt, 10, 64)
	adjustedScannedAt := int64ScannedAt + clockDrift - studentLatency - session.TeacherQRRenderingLatency

	// Validate against current RandomID
	if session.CurrentRandomID.ID == scan.ScannedRandomID {
		log.Println("CurrentID Delta: ", adjustedScannedAt-session.CurrentRandomID.ExpiredAt)
		if adjustedScannedAt-session.CurrentRandomID.ExpiredAt <= 100 {
			return true, nil
		}
		return false, errors.New("Current RandomID is invalid or expired")
	}

	// Validate against past RandomIDs
	for _, pastID := range slices.Backward(session.PastRandomIDs) {
		log.Println("PastID Delta: ", adjustedScannedAt-pastID.ExpiredAt)
		if pastID.ID == scan.ScannedRandomID {
			if adjustedScannedAt-pastID.ExpiredAt <= 100 {
				return true, nil
			}
			return false, errors.New("Past RandomID is invalid or expired")
		}
	}

	return false, errors.New("Scanned RandomID is not valid for this session")
}

func MarkStudentPresent(sessionID uint32, srn string) error {
	SessionsMutex.Lock()
	defer SessionsMutex.Unlock()

	session, exists := Sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %d not found", sessionID)
	}

	// Update the student's presence
	updated := false
	for i, student := range session.Students {
		if student.SRN == srn {
			// Only update if status actually changes
			if !session.Students[i].IsPresent {
				session.Students[i].IsPresent = true
				updated = true
			}
			break
		}
	}

	if updated {
		// Save back the updated session
		Sessions[sessionID] = session
		
		// Notify about the change in a separate goroutine to avoid blocking
		go notifyAttendanceChange(sessionID)
	}
	
	return nil
}

func abs(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

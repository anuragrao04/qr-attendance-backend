package sessions

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/anuragrao04/qr-attendance-backend/models"
)

func ValidateScan(scan models.ScanMessage, clockDrift int64) (bool, error) {
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
	adjustedScannedAt := int64ScannedAt - clockDrift - session.TeacherClockDrift
	now := time.Now().UnixMilli()

	// Validate against current RandomID
	if session.CurrentRandomID.ID == scan.ScannedRandomID {
		if abs(adjustedScannedAt-now) <= 100 {
			return true, nil
		}
		return false, errors.New("Current RandomID is invalid or expired")
	}

	// Validate against past RandomIDs
	for _, pastID := range session.PastRandomIDs {
		if pastID.ID == scan.ScannedRandomID {
			if abs(adjustedScannedAt-pastID.ExpiredAt) <= 100 {
				return true, nil
			}
			log.Println("Delta: ", adjustedScannedAt-pastID.ExpiredAt)
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
	for i, student := range session.Students {
		if student.SRN == srn {
			session.Students[i].IsPresent = true
			Sessions[sessionID] = session // Save back the updated session
			return nil
		}
	}

	return fmt.Errorf("student SRN %s not found in session %d", srn, sessionID)
}

func abs(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/anuragrao04/qr-attendance-backend/models"
	"github.com/anuragrao04/qr-attendance-backend/sessions"
	"github.com/gin-gonic/gin"
)

func StudentScan(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	SRN, err := c.Cookie("SRN")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SRN cookie not found"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to establish WebSocket connection"})
		return
	}
	defer conn.Close()

	serverBeforeTime := time.Now().UnixMilli()

	// Receive initial timestamp from the client for clock drift calculation
	var initMessage struct {
		ClientTime string `json:"clientTime"` // Unix timestamp in milliseconds
	}

	err = conn.ReadJSON(&initMessage)

	serverTime := time.Now().UnixMilli()

	studentLatency := serverTime - serverBeforeTime

	if err != nil {
		log.Printf("Failed to read initial client message: %v", err)
		conn.WriteJSON(gin.H{"status": "error", "message": "Failed to read initial data"})
		return
	}

	// Calculate clock drift
	int64ClientTime, _ := strconv.ParseInt(initMessage.ClientTime, 10, 64)
	clockDrift := serverTime - int64ClientTime // Positive means client's clock is behind

	log.Printf("Clock drift for SRN %s: %d ms", SRN, clockDrift)
	log.Printf("Latency for SRN %s: %d ms", SRN, studentLatency)

	// Handle QR code scans
	var scanMessage models.ScanMessage
	for {
		err := conn.ReadJSON(&scanMessage)
		if err != nil {
			// Handle disconnection gracefully
			log.Printf("Client disconnected or error reading message: %v", err)
			break
		}
		scanMessage.SRN = SRN // work around. because initially, SRN was sent via the scan message. Now it's through cookie

		// Validate the scanned data
		isValid, err := sessions.ValidateScan(scanMessage, clockDrift, studentLatency)
		if isValid {
			log.Println(scanMessage.SRN, "being marked present")
			// Mark student as present
			err := sessions.MarkStudentPresent(scanMessage.SessionID, scanMessage.SRN)
			if err != nil {
				log.Printf("Failed to mark student present: %v", err)
				conn.WriteJSON(gin.H{"status": "error", "message": "Failed to mark attendance"})
				continue
			}

			// Respond with success and close the connection
			conn.WriteJSON(gin.H{"status": "OK", "message": "Attendance marked successfully"})
			break
		} else {
			// Respond with failure, keep the connection open for retries
			errorMessage := err.Error()
			log.Println(scanMessage.SRN, errorMessage)
			conn.WriteJSON(gin.H{"status": "error", "message": errorMessage})
		}
	}
}

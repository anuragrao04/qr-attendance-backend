package handlers

import (
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/anuragrao04/qr-attendance-backend/models"
	"github.com/anuragrao04/qr-attendance-backend/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:3000" || origin == "https://attendance.anuragrao.site" {
			return true
		}
		log.Println("Invalid Origin:", origin)
		return false
	},
}

func CreateSession(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to establish WebSocket connection"})
		return
	}
	defer conn.Close()

	// Receive initial timestamp from the client for clock drift calculation

	conn.WriteJSON(models.RandomID{
		ID: 1234567890, // dummy random ID to probe the render latency
	})
	beforeProbe := time.Now().UnixNano()
	var initMessage struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	err = conn.ReadJSON(&initMessage)
	afterProbe := time.Now().UnixNano()

	teacherRenderLatency := (afterProbe - beforeProbe) / 2

	log.Println("Teacher Render Latency: ", teacherRenderLatency)

	if err != nil {
		log.Printf("Failed to read initial client message: %v", err)
		conn.WriteJSON(gin.H{"status": "error", "message": "Failed to read initial data"})
		return
	}

	table := c.Query("table")

	// Create a session
	log.Println("Received render time: ", teacherRenderLatency)
	sessionID, students, err := sessions.CreateSession(table, teacherRenderLatency)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer func() {
		// Clean up the session on disconnect
		sessions.DeleteSession(sessionID)
		log.Printf("Session %d cleaned up", sessionID)
	}()

	// Send the session ID to the client
	err = conn.WriteJSON(gin.H{"sessionID": sessionID, "students": students})
	if err != nil {
		log.Printf("Failed to send session ID: %v", err)
		return
	}

	// Start listening for real-time updates
	tickerRandomID := time.NewTicker(200 * time.Millisecond)
	defer tickerRandomID.Stop()

	tickerAbsentees := time.NewTicker(5 * time.Second)
	defer tickerAbsentees.Stop()

	var lastSentAbsentees []models.StudentInASession
	for {
		select {
		case <-tickerRandomID.C:
			// Generate a new random ID and update the session
			randomID := generateRandomID()
			err := sessions.UpdateRandomID(sessionID, randomID)
			if err != nil {
				log.Printf("Failed to update session random ID: %v", err)
				return
			}

			// Transmit the new random ID to the client
			err = conn.WriteJSON(randomID)
			if err != nil {
				log.Printf("Client disconnected: %v", err)
				return
			}

		case <-tickerAbsentees.C:
			// Fetch the updated absentee list every 5 seconds
			absentees, presentees, err := sessions.GetAttendanceList(sessionID)
			if err != nil {
				log.Printf("Failed to get absentees: %v", err)
				return
			}

			// Only send the absentee list if it has changed
			if !isSameAbsenteeList(lastSentAbsentees, absentees) {
				// sort the absentees by SRN

				sort.Slice(absentees, func(i, j int) bool {
					last3i, _ := strconv.Atoi(absentees[i].SRN[len(absentees[i].SRN)-3:])
					last3j, _ := strconv.Atoi(absentees[j].SRN[len(absentees[j].SRN)-3:])
					return last3i < last3j
				})
				sort.Slice(presentees, func(i, j int) bool {
					last3i, _ := strconv.Atoi(presentees[i].SRN[len(presentees[i].SRN)-3:])
					last3j, _ := strconv.Atoi(presentees[j].SRN[len(presentees[j].SRN)-3:])
					return last3i < last3j
				})

				err = conn.WriteJSON(gin.H{"absentees": absentees, "presentees": presentees})
				if err != nil {
					log.Printf("Client disconnected during absentee update: %v", err)
					return
				}
				lastSentAbsentees = absentees // Update the last sent list
			}
		}
	}

}

// Generate a new random ID
func generateRandomID() models.RandomID {
	now := time.Now().UnixMilli() // Get current time in milliseconds
	return models.RandomID{
		ID:        uint32(rand.Uint32()),
		CreatedAt: now,
		ExpiredAt: now + 200, // ID is valid for 500 ms
	}
}

func isSameAbsenteeList(a, b []models.StudentInASession) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].SRN != b[i].SRN || a[i].IsPresent != b[i].IsPresent {
			return false
		}
	}

	return true
}

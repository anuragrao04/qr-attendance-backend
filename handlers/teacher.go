package handlers

import (
	"log"
	"math/rand"
	"net/http"
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

	// Get the classroom table name from the request
	table := c.Query("table")
	log.Println("Table:", table)

	// Create a session
	sessionID, err := sessions.CreateSession(table)
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
	err = conn.WriteJSON(gin.H{"sessionID": sessionID})
	if err != nil {
		log.Printf("Failed to send session ID: %v", err)
		return
	}

	// Generate random IDs and transmit them every 200 ms
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Generate a new random ID
			randomID := generateRandomID()

			// Update the session with the new random ID
			err := sessions.UpdateRandomID(sessionID, randomID)
			if err != nil {
				log.Printf("Failed to update session random ID: %v", err)
				return
			}

			// Transmit the new random ID to the client
			err = conn.WriteJSON(randomID)
			if err != nil {
				// Detect broken pipe or disconnection
				log.Printf("Client disconnected: %v", err)
				return
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

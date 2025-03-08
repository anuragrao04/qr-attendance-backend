package handlers

import (
	"context"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
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

	// Initial latency calibration
	conn.WriteJSON(models.RandomID{
		ID: 1234567890, // dummy random ID to probe the render latency
	})

	beforeProbe := time.Now().UnixMilli()
	var initMessage struct {
		Type    string `json:"type"`
		Message int64  `json:"message"`
	}
	err = conn.ReadJSON(&initMessage)
	afterProbe := time.Now().UnixMilli()
	teacherCommunicationLatency := (afterProbe - beforeProbe) / 2

	// Read rendering time
	err = conn.ReadJSON(&initMessage)
	if err != nil {
		log.Printf("Failed to read initial client message: %v", err)
		conn.WriteJSON(gin.H{"status": "error", "message": "Failed to read initial data"})
		return
	}

	TotalRenderingLatency := teacherCommunicationLatency + initMessage.Message

	table := c.Query("table")
	sessionID, students, err := sessions.CreateSession(table, TotalRenderingLatency)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Register for attendance change events
	attendanceEvents := sessions.RegisterForAttendanceChanges(sessionID)

	// Make sure to clean up when we're done
	defer func() {
		sessions.UnregisterFromAttendanceChanges(sessionID)
		sessions.DeleteSession(sessionID)
		log.Printf("Session %d cleaned up", sessionID)
	}()

	// Send the session ID to the client
	err = conn.WriteJSON(gin.H{"sessionID": sessionID, "students": students})
	if err != nil {
		log.Printf("Failed to send session ID: %v", err)
		return
	}

	// Mutex for WebSocket writes to prevent concurrent access
	var wsWriteMutex sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Goroutine for sending random IDs
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				randomID := generateRandomID()
				err := sessions.UpdateRandomID(sessionID, randomID)
				if err != nil {
					log.Printf("Failed to update session random ID: %v", err)
					cancel()
					return
				}

				wsWriteMutex.Lock()
				err = conn.WriteJSON(randomID)
				wsWriteMutex.Unlock()

				if err != nil {
					log.Printf("Failed to send random ID: %v", err)
					cancel()
					return
				}
			}
		}
	}()

	// 2. Goroutine for reading toggle requests
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var message struct {
					Type string `json:"type"`
					SRN  string `json:"srn"`
				}

				err := conn.ReadJSON(&message)

				if err != nil {
					// If it's a timeout, just continue
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						continue
					}

					// Handle other errors
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
						log.Printf("WebSocket read error: %v", err)
					}
					cancel()
					return
				}

				// Process toggle request
				if message.Type == "TOGGLE_ATTENDANCE" && message.SRN != "" {
					log.Printf("Toggling attendance for SRN: %s in session: %d", message.SRN, sessionID)

					err := sessions.ToggleStudentAttendance(sessionID, message.SRN)
					if err != nil {
						log.Printf("Failed to toggle attendance: %v", err)

						wsWriteMutex.Lock()
						conn.WriteJSON(gin.H{"status": "error", "message": err.Error()})
						wsWriteMutex.Unlock()
					} else {
						wsWriteMutex.Lock()
						conn.WriteJSON(gin.H{"status": "OK", "message": "Attendance toggled successfully"})
						wsWriteMutex.Unlock()
					}
				}
			}
		}
	}()

	// 3. Main loop to listen for attendance change events
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-attendanceEvents:
			if !ok {
				// Channel closed
				log.Printf("Attendance event channel closed for session %d", sessionID)
				return
			}

			// Sort the lists
			sort.Slice(event.Absentees, func(i, j int) bool {
				last3i, _ := strconv.Atoi(event.Absentees[i].SRN[len(event.Absentees[i].SRN)-3:])
				last3j, _ := strconv.Atoi(event.Absentees[j].SRN[len(event.Absentees[j].SRN)-3:])
				return last3i < last3j
			})

			sort.Slice(event.Presentees, func(i, j int) bool {
				last3i, _ := strconv.Atoi(event.Presentees[i].SRN[len(event.Presentees[i].SRN)-3:])
				last3j, _ := strconv.Atoi(event.Presentees[j].SRN[len(event.Presentees[j].SRN)-3:])
				return last3i < last3j
			})

			// Send updated lists to client
			wsWriteMutex.Lock()
			err := conn.WriteJSON(gin.H{
				"type":       "ATTENDANCE_UPDATE",
				"absentees":  event.Absentees,
				"presentees": event.Presentees,
			})
			wsWriteMutex.Unlock()

			if err != nil {
				log.Printf("Failed to send attendance lists: %v", err)
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
		ExpiredAt: now + 200, // ID is valid for 200 ms
	}
}

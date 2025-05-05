package websocket2

import (
	"fmt"
	"log"
	"net/http" // For conversion
	"strconv"
	"sync"
	"time"
	"yuval/inits"
	"yuval/models"
	"yuval/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

type Hub struct {
	clients       map[*websocket.Conn]bool
	broadcast     chan string
	register      chan *websocket.Conn
	unregister    chan *websocket.Conn
	mu            sync.Mutex
	sessionClient map[uint][]*websocket.Conn // Use uint for session ID
}

var hub = Hub{
	clients:       make(map[*websocket.Conn]bool),
	broadcast:     make(chan string),
	register:      make(chan *websocket.Conn),
	unregister:    make(chan *websocket.Conn),
	sessionClient: make(map[uint][]*websocket.Conn),
}

func HandleConnections(c *gin.Context) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		log.Println("User is not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userIDStr := fmt.Sprintf("%v", userIDInterface)
	userIDUint, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("Invalid user ID format:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	var user models.User
	if err := inits.DB.Where("id = ?", userIDUint).First(&user).Error; err != nil {
		log.Println("User not found:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var userSession models.UserSession
	if err := inits.DB.Where("user_id = ? AND left_at IS NULL", userIDUint).First(&userSession).Error; err != nil {
		log.Println("User is not part of an active session:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User is not part of any active session"})
		return
	}

	sessionID := userSession.SessionID

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	hub.register <- conn

	hub.mu.Lock()
	hub.sessionClient[sessionID] = append(hub.sessionClient[sessionID], conn)
	hub.mu.Unlock()

	defer func() {
		log.Printf("Cleaning up WebSocket for user %d in session %d\n", userIDUint, sessionID)

		BroadcastMessage(sessionID, fmt.Sprintf("User %d has left the session %d", userIDUint, sessionID))

		hub.unregister <- conn

		hub.mu.Lock()
		// Remove this connection from the session's clients
		clients := hub.sessionClient[sessionID]
		for i, client := range clients {
			if client == conn {
				hub.sessionClient[sessionID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		hub.mu.Unlock()

		// Mark user as having left the session
		var existingSession models.UserSession
		if err := inits.DB.Where("user_id = ? AND session_id = ? AND left_at IS NULL", userIDUint, sessionID).First(&existingSession).Error; err == nil {
			existingSession.LeftAt = uint(time.Now().Unix())
			if err := inits.DB.Save(&existingSession).Error; err != nil {
				log.Println("Failed to update user session:", err)
			} else {
				log.Printf("Marked user %d as left session %d\n", userIDUint, sessionID)
			}
		}

		//  Check the DB for any remaining users in the session
		var count int64
		if err := inits.DB.Model(&models.UserSession{}).
			Where("session_id = ? AND left_at IS NULL", sessionID).
			Count(&count).Error; err != nil {
			log.Println("Failed to count users in session:", err)
		} else if count == 0 {
			// No users remaining in the session, mark it as inactive
			var session models.Session
			if err := inits.DB.First(&session, sessionID).Error; err == nil {
				session.Status = "inactive"
				if err := inits.DB.Save(&session).Error; err != nil {
					log.Println("Failed to mark session as inactive:", err)
				} else {
					log.Printf("Session %d marked as inactive\n", sessionID)
					go HandleSessionEnd(sessionID)
				}
			} else {
				log.Println("Failed to find session for marking inactive:", err)
			}
		}

		conn.Close()
	}()

	// Read messages loop
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			break // triggers defer
		}
		log.Printf("Received message: %s", msg)
	}
}

// HandleMessages listens for messages and broadcasts them
func HandleMessages() {
	for {
		select {
		case message := <-hub.broadcast:
			hub.mu.Lock()
			for _, clients := range hub.sessionClient {
				for _, client := range clients {
					err := client.WriteMessage(websocket.TextMessage, []byte(message))
					if err != nil {
						log.Println("WebSocket write error:", err)
						client.Close()
						delete(hub.clients, client)
					}
				}
			}
			hub.mu.Unlock()
		case client := <-hub.register:
			hub.mu.Lock()
			hub.clients[client] = true
			hub.mu.Unlock()
		case client := <-hub.unregister:
			hub.mu.Lock()
			delete(hub.clients, client)
			hub.mu.Unlock()
		}
	}
}

// BroadcastMessage sends a message to all clients in the same session
func BroadcastMessage(sessionID uint, message string) {
	hub.mu.Lock()
	for _, client := range hub.sessionClient[sessionID] {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("WebSocket write error:", err)
			client.Close()
			delete(hub.clients, client)
		}
	}
	hub.mu.Unlock()
	fmt.Println("Broadcasted to session:", sessionID, "message:", message)
}

func HandleSessionEnd(sessionID uint) {
	// Perform cleanup, analytics, logging, etc.
	log.Printf("Performing one-time cleanup for ended session %d\n", sessionID)
	utils.ConvertSessionDashToMP4(sessionID)
}

package websocket2

import (
	"fmt"
	"log"
	"net/http" // For conversion
	"strconv"
	"sync"
	"yuval/inits"
	"yuval/models"

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
	// Get the user ID from the request context
	userIDInterface, exists := c.Get("userID")
	if !exists {
		log.Println("User is not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert the user ID to a string and then to uint
	userIDStr := fmt.Sprintf("%v", userIDInterface)
	var user models.User
	if err := inits.DB.Where("id = ?", userIDStr).First(&user).Error; err != nil {
		log.Println("User not found:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Convert userID to uint
	userIDUint, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		log.Println("Invalid user ID format:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Now, retrieve the session ID for this user
	var userSession models.UserSession
	if err := inits.DB.Where("user_id = ? AND left_at IS NULL", userIDUint).First(&userSession).Error; err != nil {
		log.Println("User is not part of an active session:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User is not part of any active session"})
		return
	}

	// Get session ID from user session
	sessionID := userSession.SessionID

	// Upgrade to WebSocket connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// Register the connection for this session
	hub.register <- conn
	hub.mu.Lock()
	hub.sessionClient[sessionID] = append(hub.sessionClient[sessionID], conn)
	hub.mu.Unlock()

	defer func() {
		hub.unregister <- conn
		hub.mu.Lock()
		// Remove client from the session map
		for i, client := range hub.sessionClient[sessionID] {
			if client == conn {
				hub.sessionClient[sessionID] = append(hub.sessionClient[sessionID][:i], hub.sessionClient[sessionID][i+1:]...)
				break
			}
		}
		hub.mu.Unlock()
		conn.Close()
	}()

	// Handle incoming messages (optional)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			break
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

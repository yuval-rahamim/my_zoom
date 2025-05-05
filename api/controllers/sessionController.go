package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
	"yuval/inits"
	"yuval/models"
	"yuval/websocket2"

	"github.com/gin-gonic/gin"
)

// Function to generate a multicast address based on user ID
func GenerateMulticastIP(userID uint) string {
	baseIP := [4]int{235, 0, 0, 0}

	baseIP[2] += int(userID / 255)
	baseIP[3] += int(userID % 255)
	return fmt.Sprintf("%d.%d.%d.%d", baseIP[0], baseIP[1], baseIP[2], baseIP[3])
}

// Check if the user is part of the session
func IsUserInSession(sessionID uint, userID uint) (bool, error) {
	var userSession models.UserSession
	if err := inits.DB.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&userSession).Error; err != nil {
		return false, fmt.Errorf("user is not part of this session")
	}
	return true, nil
}

// GetSessionByUserID checks which session the user is in and returns the session ID and mcAddr
func GetSessionByUserID(userID uint) (uint, string, error) {
	var userSession models.UserSession

	// Find the user-session entry for the given userID where left_at is NULL (i.e., the user has not left the session)
	if err := inits.DB.Where("user_id = ? AND left_at IS NULL", userID).First(&userSession).Error; err != nil {
		// If no session is found or the user has left the session, return an error
		return 0, "", fmt.Errorf("user is not part of any active session")
	}

	// Retrieve the session to get mcAddr
	var session models.Session
	if err := inits.DB.Where("id = ?", userSession.SessionID).First(&session).Error; err != nil {
		// If session is not found, return an error
		return 0, "", fmt.Errorf("session not found")
	}

	// Return the session ID and mcAddr
	return session.ID, session.McAddr, nil
}

// Get all users in a session
func GetUsersInSession(sessionID uint) ([]models.UserSession, error) {
	var userSessions []models.UserSession
	if err := inits.DB.Where("session_id = ?", sessionID).Find(&userSessions).Error; err != nil {
		return nil, fmt.Errorf("error fetching users in session: %v", err)
	}
	return userSessions, nil
}

// ensures that the user is exist and convert the user id into uint
func GetValidUserID(c *gin.Context) (uint, error) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		return 0, fmt.Errorf("Unauthorized")
	}

	userIDStr := fmt.Sprintf("%v", userIDInterface)
	var user models.User
	if err := inits.DB.Where("id = ?", userIDStr).First(&user).Error; err != nil {
		return 0, fmt.Errorf("Unauthorized")
	}

	userIDUint, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID format")
	}

	return uint(userIDUint), nil
}

// GetSessionDetails fetches session details and participants
func GetSessionDetails(c *gin.Context) {
	sessionID := c.Param("id")

	// Check if session exists
	var session models.Session
	if err := inits.DB.Where("name = ?", sessionID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Check if the user is part of the session
	userID, err := GetValidUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	var userSession models.UserSession
	if err := inits.DB.Where("user_id = ? AND session_id = ?", userID, session.ID).First(&userSession).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not in this session"})
		return
	}

	// Fetch participants (user sessions) where left_at is NULL (still active)
	var participants []models.UserSession
	if err := inits.DB.Where("session_id = ? AND left_at IS NULL", session.ID).Find(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants"})
		return
	}

	// Prepare a list to hold the response data
	var response []map[string]interface{}

	// Add the current user as the first item
	var currentUser models.User
	if err := inits.DB.Where("id = ?", userID).First(&currentUser).Error; err == nil {
		// Construct stream URL for the current user
		currentUserStreamURL := fmt.Sprintf("http://localhost:3000/uploads/%d/%d/dash/stream.mpd", session.ID, userID)
		response = append(response, map[string]interface{}{
			"streamURL": currentUserStreamURL,
			"name":      currentUser.Name,
			"id":        currentUser.ID,
		})
	}

	// Fetch and add other participants
	for _, participant := range participants {
		// Skip the current user as they've already been added
		if participant.UserID == userID {
			continue
		}

		var user models.User
		if err := inits.DB.Where("id = ?", participant.UserID).First(&user).Error; err != nil {
			continue // Skip if user not found
		}

		// Construct stream URL for each participant
		streamURL := fmt.Sprintf("http://localhost:3000/uploads/%d/%d/dash/stream.mpd", session.ID, participant.UserID)

		// Add the participant to the response
		response = append(response, map[string]interface{}{
			"streamURL": streamURL,
			"name":      user.Name,
			"id":        user.ID,
		})
	}

	// Respond with session details, including stream URLs and user details
	c.JSON(http.StatusOK, gin.H{
		"session_id":   session.ID,
		"participants": response, // This is the new list with the current user first
	})
}

func DeleteUserSessionCurrent(userID uint, sessionID uint) error {
	var existingUserSession models.UserSession

	// Check if the user is already in another session
	if err := inits.DB.Where("user_id = ? AND left_at IS NULL", userID).First(&existingUserSession).Error; err == nil {
		// User is in an active session, force them to leave
		existingUserSession.LeftAt = uint(time.Now().Unix())
		if err := inits.DB.Save(&existingUserSession).Error; err != nil {
			return fmt.Errorf("failed to leave previous session: %v", err)
		}
	}
	return nil
}

// CreateUserSession ensures that a user is not in another session before joining.
func CreateUserSession(userID uint, sessionID uint) error {
	err := DeleteUserSessionCurrent(userID, sessionID)
	if err != nil {
		return err
	}
	// Create new user session
	newUserSession := models.UserSession{
		UserID:    userID,
		SessionID: sessionID,
		JoinedAt:  uint(time.Now().Unix()),
	}

	if err := inits.DB.Create(&newUserSession).Error; err != nil {
		return fmt.Errorf("failed to create user session: %v", err)
	}

	return nil
}

// CreateSession handles the creation of a new session.
func CreateSession(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := GetValidUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	// Initially, create a session without the multicast address
	session := models.Session{
		Name:   input.Name,
		HostID: userID,
		Status: "active",
	}

	// Create the session in the database to generate an ID (id should auto-increment)
	if err := inits.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Now that the session has an ID, generate and assign the multicast address
	var existingSession models.Session
	for {
		// Generate the multicast IP based on the session ID
		mcAddr := GenerateMulticastIP(session.ID)
		// Check if this address is already in use
		result := inits.DB.Where("mc_addr = ?", mcAddr).First(&existingSession)
		if result.Error != nil { // No existing session with the same multicast address
			session.McAddr = mcAddr
			break
		}
	}

	// Update the session with the multicast address
	if err := inits.DB.Save(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session with multicast address"})
		return
	}

	// Create the user-session link
	if err := CreateUserSession(userID, session.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Respond with the session creation success
	c.JSON(http.StatusOK, gin.H{"message": "Session created successfully", "session_id": session.ID})
}

// JoinSession allows a user to join a session.
func JoinSession(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := GetValidUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	var session models.Session
	if err := inits.DB.Where("name = ?", input.Name).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	if err := CreateUserSession(userID, session.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast to all clients in the session that a new user has joined
	message := fmt.Sprintf("User %d has joined the session %s", userID, session.Name)
	websocket2.BroadcastMessage(session.ID, message)

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined the session", "session_id": session.ID})
}

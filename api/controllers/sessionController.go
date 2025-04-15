package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	"yuval/inits"
	"yuval/models"
	"yuval/utils"
	"yuval/websocket"

	"github.com/gin-gonic/gin"
)

var dashRunning = make(map[uint]bool)
var dashMutex = &sync.Mutex{}

// Function to generate a multicast address based on session ID
func generateMulticastIP(sessionID uint) string {
	return fmt.Sprintf("239.255.255.%d", sessionID)
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

// CreateUserSession ensures that a user is not in another session before joining.
func CreateUserSession(userID uint, sessionID uint) error {
	var existingUserSession models.UserSession

	// Check if the user is already in another session
	if err := inits.DB.Where("user_id = ? AND left_at IS NULL", userID).First(&existingUserSession).Error; err == nil {
		// User is in an active session, force them to leave
		existingUserSession.LeftAt = uint(time.Now().Unix())
		if err := inits.DB.Save(&existingUserSession).Error; err != nil {
			return fmt.Errorf("failed to leave previous session: %v", err)
		}
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
		mcAddr := generateMulticastIP(session.ID)
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
	websocket.BroadcastMessage(session.ID, message)

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined the session", "session_id": session.ID})
}

// ConvertToMPEGTS handles real-time video slices (e.g., WebM chunks) and multicasts them as MPEG-TS
func ConvertToMPEGTS(c *gin.Context) {
	userID, err := GetValidUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	sessionID, _, err := GetSessionByUserID(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Auto-start MPEG-DASH conversion for this session if not already running
	dashMutex.Lock()
	if !dashRunning[sessionID] {
		dashRunning[sessionID] = true
		go func() {
			ctx, _ := gin.CreateTestContext(nil)
			ctx.Request = c.Request
			ctx.Set("userID", userID) // Inject user ID
			ConvertToMPEGDASH(ctx)
		}()
	}
	dashMutex.Unlock()

	// Parse uploaded WebM slice
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video slice upload"})
		return
	}

	// Define storage paths
	sessionPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID))
	userPath := filepath.Join(sessionPath, fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(userPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directories"})
		return
	}

	// Save slice temporarily
	timestamp := time.Now().UnixNano()
	sliceFileName := fmt.Sprintf("slice-%d.webm", timestamp)
	slicePath := filepath.Join(userPath, sliceFileName)
	if err := c.SaveUploadedFile(file, slicePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save slice"})
		return
	}

	// Multicast this slice as MPEG-TS
	multicastIp := generateMulticastIP(sessionID)
	cmd := fmt.Sprintf(
		"ffmpeg -y -i %s -c:v libx264 -c:a aac -b:a 160k -bsf:v h264_mp4toannexb -f mpegts udp://%s:555?pkt_size=1316",
		slicePath, multicastIp,
	)

	go func() {
		if err := utils.RunCommand(cmd); err != nil {
			websocket.BroadcastMessage(sessionID, fmt.Sprintf("Failed to convert slice for user %d", userID))
			return
		}
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Slice received and processing started"})
}

// ConvertToMPEGDASH processes the MPEG-TS file to MPEG-DASH locally
func ConvertToMPEGDASH(c *gin.Context) {
	// Retrieve the userID from the context (e.g., from a session or JWT)
	userID, err := GetValidUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	// Retrieve sessionID from the context or database
	sessionID, _, err := GetSessionByUserID(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Define paths for the TS file (MPEG-TS should have been created previously)
	sessionPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID))
	userPath := filepath.Join(sessionPath, fmt.Sprintf("%d", userID))

	// Acknowledge the start of video processing
	message := fmt.Sprintf("Started processing video for user %d. Please wait...", userID)
	websocket.BroadcastMessage(sessionID, message)

	// Set retry parameters
	maxRetries := 5
	retryInterval := 10 * time.Second
	timeout := 2 * time.Minute // Maximum waiting time for the conversion process

	// Start a timeout timer
	timeoutChan := time.After(timeout)
	retryCount := 0

	for {
		// Check if the timeout has been reached
		select {
		case <-timeoutChan:
			// Notify user that the conversion timed out
			websocket.BroadcastMessage(sessionID, fmt.Sprintf("MPEG-DASH conversion timed out for user %d", userID))
			c.JSON(http.StatusRequestTimeout, gin.H{
				"message": "Conversion timed out. Please try again later.",
			})
			return
		default:
			// Try the conversion locally
			dashOutputDir := filepath.Join(userPath, "dash")
			if err := os.MkdirAll(dashOutputDir, os.ModePerm); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create DASH directory"})
				return
			}

			multicastIp := generateMulticastIP(sessionID)
			// cmd := fmt.Sprintf("ffmpeg -i %s -map 0 -codec:v libx264 -b:v 1000k -codec:a aac -b:a 128k -f dash -seg_duration 20 -use_template 1 -use_timeline 1 %s/stream.mpd", mpegTSPath, dashOutputDir)
			cmd := fmt.Sprintf("ffmpeg -i udp://%s:555 -map 0 -codec:v libx264 -b:v 1000k -codec:a aac -b:a 128k -f dash -seg_duration 20 -use_template 1 -use_timeline 1 %s/stream.mpd", multicastIp, dashOutputDir)
			// Run the conversion command
			err := utils.RunCommand(cmd)
			if err == nil {
				// If successful, broadcast the success message
				successMessage := fmt.Sprintf("Video processed to MPEG-DASH and is saved locally for user %d.", userID)
				websocket.BroadcastMessage(sessionID, successMessage)

				// Notify frontend that MPEG-DASH is ready
				websocket.BroadcastMessage(sessionID, "MPEG-DASH Ready")

				// Immediate response indicating the process is ongoing
				c.JSON(http.StatusOK, gin.H{
					"message": "Video processing started. Please wait for the conversion to complete.",
				})
				return
			}

			// If the command failed, increment the retry count
			retryCount++
			if retryCount >= maxRetries {
				// Notify user if maximum retries are reached
				websocket.BroadcastMessage(sessionID, fmt.Sprintf("Error: Maximum retries reached for MPEG-DASH conversion for user %d", userID))
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": fmt.Sprintf("Error: Maximum retries reached for MPEG-DASH conversion for user %d", userID),
				})
				return
			}

			// Retry after a delay
			websocket.BroadcastMessage(sessionID, fmt.Sprintf("Retrying conversion (attempt %d)...", retryCount))
			time.Sleep(retryInterval)
		}
	}
}

// ServeDashFile serves the DASH manifest file (.mpd) to the client
func ServeDashFile(c *gin.Context) {
	var requestData struct {
		SessionID uint   `json:"sessionID" binding:"required"`
		UserID    uint   `json:"userID" binding:"required"`
		FileName  string `json:"fileName" binding:"required"`
	}

	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Construct the file path based on the sessionID, userID, and fileName
	filePath := filepath.Join("./uploads", fmt.Sprintf("%d", requestData.SessionID), fmt.Sprintf("%d", requestData.UserID), "dash", requestData.FileName)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Serve the file
	c.File(filePath)
}

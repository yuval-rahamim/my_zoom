package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"yuval/inits"
	"yuval/models"
	"yuval/utils"

	"github.com/gin-gonic/gin"
)

// Check if the user is part of the session
func IsUserInSession(sessionID uint, userID uint) (bool, error) {
	var userSession models.UserSession
	if err := inits.DB.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&userSession).Error; err != nil {
		return false, fmt.Errorf("user is not part of this session")
	}
	return true, nil
}

// GetSessionByUserID checks which session the user is in and returns the session or session ID
func GetSessionByUserID(userID uint) (uint, error) {
	var userSession models.UserSession

	// Find the user-session entry for the given userID where left_at is NULL (i.e., the user has not left the session)
	if err := inits.DB.Where("user_id = ? AND left_at IS NULL", userID).First(&userSession).Error; err != nil {
		// If no session is found or the user has left the session, return an error
		return 0, fmt.Errorf("user is not part of any active session")
	}

	// Return the session ID
	return userSession.SessionID, nil
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

	session := models.Session{
		Name:   input.Name,
		HostID: userID,
		Status: "active",
	}

	if err := inits.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := CreateUserSession(userID, session.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined the session", "session_id": session.ID})
}

// Convert MP4 (or live stream) to MPEG-TS, then convert to MPEG-DASH
func ConvertToMPEGTS(c *gin.Context) {
	userID, err := GetValidUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}

	sessionID, err := GetSessionByUserID(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	// Parse file from request
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file upload"})
		return
	}

	// Create session and user directories if they do not exist
	sessionPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID))
	userPath := filepath.Join(sessionPath, fmt.Sprintf("%d", userID))

	// Create the session and user directories
	if err := os.MkdirAll(userPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	// Save uploaded file
	filePath := filepath.Join(userPath, file.Filename)
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Convert MP4 to MPEG-TS
	mpegTSPath := filepath.Join(userPath, "output.ts")
	cmd := fmt.Sprintf("ffmpeg -i %s -c:v libx264 -c:a aac -b:a 160k -bsf:v h264_mp4toannexb -f mpegts -crf 32 udp://235.235.235.235:555", filePath)

	if err := utils.RunCommand(cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FFmpeg TS conversion failed"})
		return
	}

	// Convert MPEG-TS to MPEG-DASH
	convertToMPEGDASH(mpegTSPath, sessionID, userID, c)
}

// Convert MPEG-TS to MPEG-DASH
func convertToMPEGDASH(mpegTSPath string, sessionID uint, userID uint, c *gin.Context) {
	// Create output directory for DASH files
	mpegDashDir := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID), fmt.Sprintf("%d", userID), "dash")
	if err := os.MkdirAll(mpegDashDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create DASH directory"})
		return
	}

	mpdFilePath := filepath.Join(mpegDashDir, "stream.mpd")
	go func() {
		cmd := fmt.Sprintf("ffmpeg -i udp://235.235.235.235:555 -map 0 -codec:v libx264 -b:v 1000k -codec:a aac -b:a 128k -f dash -seg_duration 20 -use_template 1 -use_timeline 1 %s", mpdFilePath)

		if err := utils.RunCommand(cmd); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "FFmpeg DASH conversion failed"})
			return
		}

	}()
	// cmd := fmt.Sprintf("ffmpeg -i udp://235.235.235.235:555 -map 0 -codec:v libx264 -b:v 1000k -codec:a aac -b:a 128k -f dash -seg_duration 20 -use_template 1 -use_timeline 1 %s", mpdFilePath)

	// if err := utils.RunCommand(cmd); err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "FFmpeg DASH conversion failed"})
	// 	return
	// }

	// Construct the stream URL with sessionID and userID
	streamURL := fmt.Sprintf("http://localhost:3000/uploads/%d/%d/dash/stream.mpd", sessionID, userID)

	// Respond with the DASH manifest URL
	c.JSON(http.StatusOK, gin.H{
		"message":    "Video processed",
		"stream_url": streamURL,
	})
}

// ServeDashFile serves the DASH manifest file (.mpd) to the client.
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

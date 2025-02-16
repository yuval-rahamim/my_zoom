package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"yuval/inits"
	"yuval/models"
	"yuval/utils"

	"github.com/gin-gonic/gin"
)

// CreateSession handles the creation of a new session.
func CreateSession(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Retrieve the user from the session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Create the session
	session := models.Session{
		Name:   input.Name,
		HostID: userID.(uint),
		Status: "active",
	}

	if err := inits.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session created successfully", "session_id": session.ID})
}

// JoinSession allows a user to join a session.
func JoinSession(c *gin.Context) {
	var input struct {
		SessionID uint `json:"session_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Retrieve the user from the session
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Check if session exists
	var session models.Session
	if err := inits.DB.First(&session, input.SessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Add the user to the session
	userSession := models.UserSession{
		UserID:    userID.(uint),
		SessionID: input.SessionID,
		JoinedAt:  uint(time.Now().Unix()),
	}

	if err := inits.DB.Create(&userSession).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined the session"})
}

// ServeDashFile serves the DASH manifest file (.mpd) to the client.
func ServeDashFile(c *gin.Context) {
	var requestData struct {
		UserName string `json:"userName" binding:"required"`
		FileName string `json:"fileName" binding:"required"`
	}

	if err := c.BindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userName := requestData.UserName
	fileName := requestData.FileName

	// Construct the file path based on the userName and fileName
	filePath := filepath.Join("uploads", userName, "dash", fileName)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Serve the file
	c.File(filePath)
}

// Convert MP4 (or live stream) to MPEG-TS, then convert to MPEG-DASH
func ConvertToMPEGTS(c *gin.Context) {
	userName := c.PostForm("Name")

	// Parse file from request
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file upload"})
		return
	}

	// Save uploaded file
	uploadDir := filepath.Join("uploads", userName)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	filePath := filepath.ToSlash(filepath.Join(uploadDir, file.Filename))
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Convert MP4 to MPEG-TS
	mpegTSPath := filepath.ToSlash(filepath.Join(uploadDir, "output.ts"))
	cmd := fmt.Sprintf("ffmpeg -i %s -c:v libx264 -c:a aac -b:a 160k -bsf:v h264_mp4toannexb -f mpegts -crf 32 %s", filePath, mpegTSPath)

	if err := utils.RunCommand(cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FFmpeg TS conversion failed"})
		return
	}

	// Convert MPEG-TS to MPEG-DASH
	convertToMPEGDASH(mpegTSPath, userName, c)
}

// Convert MPEG-TS to MPEG-DASH
func convertToMPEGDASH(mpegTSPath string, userName string, c *gin.Context) {
	// Create output directory
	mpegDashDir := "uploads/" + userName + "/dash"
	if err := os.MkdirAll(mpegDashDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create DASH directory"})
		return
	}

	mpdFilePath := filepath.Join(mpegDashDir, "stream.mpd")
	cmd := fmt.Sprintf("ffmpeg -i %s -map 0 -codec:v libx264 -b:v 1000k -codec:a aac -b:a 128k -f dash -seg_duration 20 -use_template 1 -use_timeline 1  %s", mpegTSPath, mpdFilePath)
	if err := utils.RunCommand(cmd); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "FFmpeg DASH conversion failed"})
		return
	}

	// Construct the stream URL with userName and fileName (stream.mpd)
	streamURL := "http://localhost:3000/uploads/" + userName + "/dash/stream.mpd"

	// Respond with the DASH manifest URL
	c.JSON(http.StatusOK, gin.H{
		"message":    "Video processed",
		"stream_url": streamURL,
	})
}

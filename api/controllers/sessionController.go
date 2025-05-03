package controllers

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"yuval/inits"
	"yuval/models"
	"yuval/utils"
	"yuval/websocket2"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Represents a simplified version of the MPD XML structure
type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	Period  Period   `xml:"Period"`
}

type Period struct {
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	SegmentTemplate SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
	SegmentTimeline SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
	S []Segment `xml:"S"`
}

type Segment struct {
	T int64 `xml:"t,attr"` // start time
	D int64 `xml:"d,attr"` // duration
	R int64 `xml:"r,attr"` // repeat count (optional)
}

// Function to generate a multicast address based on user ID
func generateMulticastIP(userID uint) string {
	baseIP := [4]int{235, 0, 0, 0}
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
	websocket2.BroadcastMessage(session.ID, message)

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined the session", "session_id": session.ID})
}

// Start cleaner in background
func StartDashCleaner(dashDir string) {
	a := 0
	for {
		time.Sleep(10 * time.Second) // run every 10 seconds
		count := cleanOldSegments(dashDir)
		log.Printf("Cleaned %d old segments\n", count)
		if count == 0 {
			print("no segments to clean\n")
			a++
		}
		if a > 10 {
			print("no segments to clean for 10 times\n")
			break
		}
	}
}

// Now cleanOldSegments RETURNS an int
func cleanOldSegments(dashDir string) int {
	mpdPath := filepath.Join(dashDir, "stream.mpd")

	// Read MPD file
	data, err := os.ReadFile(mpdPath)
	if err != nil {
		return 0
	}

	var mpd MPD
	err = xml.Unmarshal(data, &mpd)
	if err != nil {
		return 0
	}

	// Build set of valid segments
	validSegments := make(map[string]struct{})

	for _, adaptationSet := range mpd.Period.AdaptationSets {
		for _, segment := range adaptationSet.SegmentTemplate.SegmentTimeline.S {
			segmentName := buildSegmentName(segment.T) // customize this if needed
			validSegments[segmentName] = struct{}{}
		}
	}

	// List all .m4s files
	files, err := os.ReadDir(dashDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".m4s") {
			if _, ok := validSegments[file.Name()]; !ok {
				err := os.Remove(filepath.Join(dashDir, file.Name()))
				if err == nil {
					count++
					log.Printf("Deleted old segment: %s\n", file.Name())
				} else {
					log.Printf("Failed to delete segment: %s, error: %v\n", file.Name(), err)
				}
			}
		}
	}

	return count
}

// Build the segment filename based on timestamp or ID
func buildSegmentName(timestamp int64) string {
	return fmt.Sprintf("chunk-stream0-%d.m4s", timestamp)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all connections
	},
}

func HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("userID")
	if userIDStr == "" {
		http.Error(w, "Missing userID", http.StatusBadRequest)
		return
	}

	userIDUint, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid userID", http.StatusBadRequest)
		return
	}
	userID := uint(userIDUint)

	sessionID, _, err := GetSessionByUserID(userID)
	if err != nil {
		http.Error(w, "Session not found: "+err.Error(), http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	multicastIP := generateMulticastIP(userID)
	udpURL := fmt.Sprintf("udp://%s:55?pkt_size=1316", multicastIP)

	cmd := exec.Command("ffmpeg",
		"-f", "webm",
		"-i", "pipe:0",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-g", "30",
		"-sc_threshold", "0",
		"-c:a", "aac",
		"-f", "mpegts",
		udpURL, // 	"udp://235.235.235.235:55?pkt_size=1316"
	)

	ffmpegIn, err := cmd.StdinPipe()
	if err != nil {
		log.Println("Failed to get ffmpeg stdin:", err)
		return
	}

	cmd.Stderr = log.Writer()
	go func() {
		err = cmd.Start()
		if err != nil {
			log.Println("Failed to start ffmpeg:", err)
			return
		}
	}()
	go func() {
		ConvertToMPEGDASH(sessionID, userID)
	}()

	log.Println("FFmpeg started, waiting for video chunks...")

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket closed:", err)
			break
		}
		_, err = ffmpegIn.Write(data)
		if err != nil {
			log.Println("Error writing to ffmpeg stdin:", err)
			break
		}
	}

	ffmpegIn.Close()
	cmd.Wait()
	log.Println("FFmpeg process exited")
}

// ConvertToMPEGDASH listens to a multicast MPEG-TS stream and converts it into MPEG-DASH segments
func ConvertToMPEGDASH(sessionID uint, userID uint) {
	// Set up output directory
	sessionPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID))
	userPath := filepath.Join(sessionPath, fmt.Sprintf("%d", userID))
	dashOutputDir := filepath.Join(userPath, "dash")

	if err := os.MkdirAll(dashOutputDir, os.ModePerm); err != nil {
		return
	}

	// FFmpeg command to listen to multicast MPEG-TS and convert to MPEG-DASH
	multicastIp := generateMulticastIP(userID)
	cmd := fmt.Sprintf(`ffmpeg -i udp://%s:55 -map 0 -codec:v libx264 -preset ultrafast -tune zerolatency -codec:a aac -b:a 128k -f dash -seg_duration 2 -use_template 1 -use_timeline 1 %s/stream.mpd`, multicastIp, dashOutputDir)

	// Run conversion in a goroutine to allow immediate HTTP response
	go func() {
		_ = utils.RunCommand(cmd)
		// Broadcast to all clients in the session that a new user has joined
		// message := fmt.Sprintf("Dash is ready for %d", userID)
		// websocket2.BroadcastMessage(sessionID, message)
	}()
	// go func() {
	// 	StartDashCleaner(dashOutputDir)
	// }()
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

package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"yuval/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

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
		"-analyzeduration", "1500000",
		"-i", "pipe:0",
		"-fflags", "nobuffer+flush_packets+discardcorrupt",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-g", "30",
		"-sc_threshold", "0",
		"-c:a", "aac",
		"-f", "mpegts",
		udpURL,
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
	cmd := fmt.Sprintf(`ffmpeg -re -i udp://%s:55 -codec:v libx264 -preset ultrafast -tune zerolatency -codec:a aac -b:a 128k -f dash -seg_duration 1 -use_template 1 -use_timeline 1 %s/stream.mpd`, multicastIp, dashOutputDir)

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

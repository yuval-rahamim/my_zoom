package dasher

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"yuval/controllers"
	"yuval/websocket2"

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

	sessionID, _, err := controllers.GetSessionByUserID(userID)
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
	sessionPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID))
	userPath := filepath.Join(sessionPath, fmt.Sprintf("%d", userID))
	dashOutputDir := filepath.Join(userPath, "dash")

	// multicastIP := controllers.GenerateMulticastIP(userID)
	// udpURL := fmt.Sprintf("udp://%s:55?localaddr=myzoom.co.il&pkt_size=1316", multicastIP)
	ch := make(chan []byte, 10500000)
	cmd := exec.Command("ffmpeg",
		"-f", "webm",
		"-analyzeduration", "100000",
		"-probesize", "32",
		"-i", "pipe:0",
		"-fflags", "nobuffer+flush_packets+discardcorrupt",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-g", "25",
		"-keyint_min", "25",
		"-sc_threshold", "0",
		"-c:a", "aac",
		"-b:a", "128k",
		"-f", "mpegts",
		dashOutputDir+"/stream.ts",
	)

	ffmpegIn, err := cmd.StdinPipe()
	if err != nil {
		log.Println("Failed to get ffmpeg stdin:", err)
		return
	}

	cmd.Stderr = log.Writer()
	go func() {
		ConvertToMPEGDASH(sessionID, userID, ch)
	}()
	go func() {
		err = cmd.Start()
		if err != nil {
			log.Println("Failed to start ffmpeg:", err)
			return
		}
	}()

	log.Println("FFmpeg started, waiting for video chunks...")

	defer ffmpegIn.Close()
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket closed:", err)
			break
		}
		ch <- data
		_, err = ffmpegIn.Write(data)
		if err != nil {
			log.Println("Error writing to ffmpeg stdin:", err)
			break
		}
	}

	cmd.Wait()
	log.Println("FFmpeg process exited")
}

// ConvertToMPEGDASH listens to a multicast MPEG-TS stream and converts it into MPEG-DASH segments
func ConvertToMPEGDASH(sessionID uint, userID uint, ch chan []byte) {
	// Set up output directory
	sessionPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID))
	userPath := filepath.Join(sessionPath, fmt.Sprintf("%d", userID))
	dashOutputDir := filepath.Join(userPath, "dash")

	if err := os.MkdirAll(dashOutputDir, os.ModePerm); err != nil {
		return
	}

	cmd := exec.Command("ffmpeg",
		"-loglevel", "quiet",
		"-re",
		"-f", "webm",
		"-analyzeduration", "100000",
		"-probesize", "32",
		"-i", "pipe:0",
		"-fflags", "nobuffer+flush_packets+discardcorrupt",
		"-codec:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-codec:a", "aac",
		"-b:a", "128k",
		"-f", "dash",
		"-fflags", "nobuffer+flush_packets+discardcorrupt",
		"-g", "25",
		"-keyint_min", "25",
		"-sc_threshold", "0",
		"-seg_duration", "1",
		"-window_size", "5",
		"-extra_window_size", "5",
		"-remove_at_exit", "1",
		dashOutputDir+"/stream.mpd",
	)

	// cmd := exec.Command("ffmpeg",
	// 	"-loglevel", "quiet",
	// 	"-re",
	// 	"-i", "pipe:0",
	// 	"-fflags", "nobuffer+flush_packets+discardcorrupt",
	// 	// Split and scale the input to two outputs
	// 	"-filter_complex", "[0:v]split=2[v1][v2];[v1]scale=w=1280:h=720[v1out];[v2]scale=w=640:h=360[v2out]",

	// 	// Map the video and audio streams
	// 	"-map", "[v1out]",
	// 	"-map", "[v2out]",
	// 	"-map", "0:a",
	// 	"-map", "0:a",

	// 	// Codec settings for both video streams
	// 	"-c:v", "libx264",
	// 	"-preset", "ultrafast",
	// 	"-tune", "zerolatency",
	// 	"-g", "25",
	// 	"-keyint_min", "25",
	// 	"-sc_threshold", "0",
	// 	"-b:v:0", "1000k", // high quality
	// 	"-b:v:1", "800k", // low quality

	// 	// Audio settings
	// 	"-c:a", "aac",
	// 	"-b:a:0", "128k",
	// 	"-b:a:1", "64k",

	// 	// DASH output
	// 	"-f", "dash",
	// 	"-fflags", "nobuffer+flush_packets+discardcorrupt",
	// 	"-seg_duration", "1",
	// 	"-use_template", "1",
	// 	"-use_timeline", "1",
	// 	"-adaptation_sets", "id=0,streams=v id=1,streams=a",
	// 	"-window_size", "5",
	// 	"-extra_window_size", "5",
	// 	"-remove_at_exit", "0",

	// 	dashOutputDir+"/stream.mpd",
	// )

	ffmpegIn, err := cmd.StdinPipe()
	if err != nil {
		log.Println("Failed to get ffmpeg stdin:", err)
		return
	}

	// Broadcast to all clients in the session that a new user has joined
	websocket2.BroadcastMessage(sessionID, "stream started")

	cmd.Stderr = log.Writer()
	go func() {
		err = cmd.Start()
		if err != nil {
			log.Println("Failed to start ffmpeg:", err)
			return
		}
	}()

	for {
		select {
		case data := <-ch:
			ffmpegIn.Write(data)
			if err != nil {
				log.Println("Error writing to ffmpeg stdin:", err)
				break
			}
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

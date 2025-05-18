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
	"yuval/utils"
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

	multicastIP := controllers.GenerateMulticastIP(userID)
	udpURL := fmt.Sprintf("udp://%s:55?localaddr=myzoom.co.il&pkt_size=1316", multicastIP)

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
		"-b:a", "128k",
		"-ac", "2",
		"-ar", "44100",
		"-f", "mpegts",
		"-fflags", "nobuffer+flush_packets+discardcorrupt",
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
	multicastIp := controllers.GenerateMulticastIP(userID)
	cmd := fmt.Sprintf(
		`ffmpeg -re -i udp://%s:55?localaddr=myzoom.co.il -fflags nobuffer+flush_packets+discardcorrupt -codec:v libx264 -preset ultrafast -tune zerolatency -codec:a aac -b:a 128k -f dash -seg_duration 1 -window_size 5 -extra_window_size 5 -remove_at_exit 0 %s/stream.mpd`,
		multicastIp, dashOutputDir,
	)

	// Broadcast to all clients in the session that a new user has joined
	websocket2.BroadcastMessage(sessionID, "stream started")
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

// ConvertToMPEGDASH listens to a multicast MPEG-TS stream and converts it into adaptive MPEG-DASH segments
// func ConvertToMPEGDASH(sessionID uint, userID uint) {
// 	// Set up output directory
// 	sessionPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID))
// 	userPath := filepath.Join(sessionPath, fmt.Sprintf("%d", userID))
// 	dashOutputDir := filepath.Join(userPath, "dash")

// 	if err := os.MkdirAll(dashOutputDir, os.ModePerm); err != nil {
// 		fmt.Printf("Failed to create directory: %v\n", err)
// 		return
// 	}

// 	multicastIp := controllers.GenerateMulticastIP(userID)

// 	args := []string{
// 		"-re",
// 		"-i", fmt.Sprintf("udp://%s:55", multicastIp),

// 		"-filter_complex", "[0:v]split=3[v1][v2][v3];[v1]scale=1280:720[v720];[v2]scale=854:480[v480];[v3]scale=426:240[v240]",

// 		"-map", "[v720]", "-c:v:0", "libx264", "-b:v:0", "1500k",
// 		"-map", "[v480]", "-c:v:1", "libx264", "-b:v:1", "800k",
// 		"-map", "[v240]", "-c:v:2", "libx264", "-b:v:2", "400k",
// 		"-map", "0:a?", "-c:a", "aac", "-b:a", "128k",

// 		"-use_timeline", "1",
// 		"-use_template", "1",
// 		"-window_size", "5",
// 		"-extra_window_size", "5",
// 		"-remove_at_exit", "0",
// 		"-adaptation_sets", "id=0,streams=v id=1,streams=a",
// 		"-init_seg_name", "init_$RepresentationID$.mp4",
// 		"-media_seg_name", "chunk_$RepresentationID$_$Number$.m4s",
// 		"-f", "dash",
// 		"stream.mpd", // relative, since we set cmd.Dir
// 	}

// 	cmd := exec.Command("ffmpeg", args...)
// 	cmd.Dir = dashOutputDir // output goes here
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	websocket2.BroadcastMessage(sessionID, "stream started")

// 	go func() {
// 		err := cmd.Run()
// 		if err != nil {
// 			fmt.Printf("FFmpeg failed: %v\n", err)
// 		}
// 	}()
// }

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

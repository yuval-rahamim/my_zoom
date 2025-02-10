package controllers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Handle video upload, conversion to MPEG-TS, and MPEG-DASH
func handleVideoUpload(c *gin.Context) {
	// Parse incoming form data (video file)
	file, _, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file"})
		return
	}
	defer file.Close()

	// Create a temporary file to store the video
	tempFile, err := os.CreateTemp("", "video_*.mp4")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create temp file"})
		return
	}
	defer os.Remove(tempFile.Name()) // Cleanup temp file on exit

	// Save the uploaded file to the temp file using io.Copy
	_, err = io.Copy(tempFile, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Get the file path of the saved file
	tempFilePath := tempFile.Name()

	// Convert the MP4 to MPEG-TS using FFmpeg
	mpegtsFilePath := filepath.Join("temp", "output.mpegts")
	err = convertToMpegts(tempFilePath, mpegtsFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to MPEG-TS"})
		return
	}

	// Convert the MPEG-TS to MPEG-DASH (using FFmpeg to segment and create a .mpd file)
	mpdFilePath := filepath.Join("temp", "output.mpd")
	err = convertToMpegDash(mpegtsFilePath, mpdFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to MPEG-DASH"})
		return
	}

	// Respond with the URL to the .mpd file
	c.JSON(http.StatusOK, gin.H{
		"mpdUrl": "http://localhost:3000/videos/output.mpd", // URL should be mapped correctly
	})
}

// Convert MP4 to MPEG-TS using FFmpeg
func convertToMpegts(inputPath, outputPath string) error {
	cmd := fmt.Sprintf("ffmpeg -i %s -c:v mpeg2video -c:a mp2 -f mpegts %s", inputPath, outputPath)
	return RunCommand(cmd)
}

// Convert MPEG-TS to MPEG-DASH using FFmpeg
func convertToMpegDash(inputPath, outputPath string) error {
	cmd := fmt.Sprintf("ffmpeg -i %s -c:v libx264 -c:a aac -f dash -dash_segment_filename %s/segment_$Number$.m4s %s", inputPath, filepath.Dir(outputPath), outputPath)
	return RunCommand(cmd)
}

// RunCommand executes a command in the shell and handles errors
func RunCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	err := cmd.Run()
	if err != nil {
		log.Printf("Command failed: %v\n", err)
		return fmt.Errorf("Command failed: %v", err)
	}
	return nil
}

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/convert", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run(":4000")
}

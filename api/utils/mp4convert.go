package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
	"yuval/inits"
	"yuval/models"
)

func ConvertSessionDashToMP4(sessionID uint) {
	time.Sleep(3 * time.Second) // run every 10 seconds
	// Load all user sessions under this session
	var userSessions []models.UserSession
	if err := inits.DB.Where("session_id = ?", sessionID).Find(&userSessions).Error; err != nil {
		log.Printf("Failed to load user sessions for session %d: %v\n", sessionID, err)
		return
	}

	// Create the VOD output folder
	vodFolder := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID), "vod")
	if err := os.MkdirAll(vodFolder, os.ModePerm); err != nil {
		log.Printf("Failed to create VOD folder for session %d: %v\n", sessionID, err)
		return
	}

	for _, us := range userSessions {
		userID := us.UserID
		mpdPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID), fmt.Sprintf("%d", userID), "dash", "stream.mpd")
		outputPath := filepath.Join(vodFolder, fmt.Sprintf("%d.mp4", userID))
		//ffmpeg -y -i %s -c copy -bsf:a aac_adtstoasc -err_detect ignore_err -fflags +discardcorrupt %s

		//ffmpeg -re -i input.mp4 -map 0:v -map 0:a -map 0:v -map 0:a -map 0:v -map 0:a -b:v:0 800k -s:v:0 640x360  -profile:v:0 main -b:v:1 1500k -s:v:1 1280x720 -profile:v:1 main -b:v:2 3000k -s:v:2 1920x1080 -profile:v:2 high -b:a 128k -c:a aac -c:v libx264 -keyint_min 48 -g 48 -sc_threshold 0 -use_timeline 1 -use_template 1 -seg_duration 4 -adaptation_sets "id=0,streams=v id=1,streams=a"
		cmd := fmt.Sprintf(
			`ffmpeg -y -i %s -c copy -bsf:a aac_adtstoasc -err_detect ignore_err -fflags +discardcorrupt %s`,
			mpdPath, outputPath)

		go func(userID uint, cmd string) {
			log.Printf("Starting MP4 conversion for user %d from DASH\n", userID)
			if err := RunCommand(cmd); err != nil {
				log.Printf("MP4 conversion failed for user %d: %v\n", userID, err)
			} else {
				log.Printf("MP4 conversion complete for user %d\n", userID)
			}
		}(userID, cmd)
	}
}

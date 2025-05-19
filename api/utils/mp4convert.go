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
	time.Sleep(15 * time.Second) // slight delay before starting conversion

	// Load all user sessions for this meeting
	var userSessions []models.UserSession
	if err := inits.DB.Where("session_id = ?", sessionID).Find(&userSessions).Error; err != nil {
		log.Printf("[ConvertSessionDashToMP4] Failed to load user sessions for session %d: %v\n", sessionID, err)
		return
	}

	// Create the VOD output folder
	vodFolder := filepath.Join("./videos", fmt.Sprintf("%d", sessionID))
	if err := os.MkdirAll(vodFolder, os.ModePerm); err != nil {
		log.Printf("[ConvertSessionDashToMP4] Failed to create VOD folder for session %d: %v\n", sessionID, err)
		return
	}

	for _, us := range userSessions {
		userID := us.UserID
		mpdPath := filepath.Join("./uploads", fmt.Sprintf("%d", sessionID), fmt.Sprintf("%d", userID), "dash", "stream.mpd")
		outputPath := filepath.Join(vodFolder, fmt.Sprintf("%d.mp4", userID))

		cmd := fmt.Sprintf(
			`ffmpeg -y -i %s -c copy -bsf:a aac_adtstoasc -ignore_unknown -err_detect ignore_err -fflags +discardcorrupt %s`,
			mpdPath, outputPath)

		go func(userID uint, cmd string) {
			log.Printf("[ConvertSessionDashToMP4] Starting MP4 conversion for user %d from DASH\n", userID)
			if err := RunCommand(cmd); err != nil {
				log.Printf("[ConvertSessionDashToMP4] MP4 conversion failed for user %d: %v\n", userID, err)
			} else {
				log.Printf("[ConvertSessionDashToMP4] MP4 conversion complete for user %d\n", userID)
			}
		}(userID, cmd)
	}
}

package utils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"yuval/inits"
	"yuval/models"
)

func ConvertSessionTsToMP4(sessionID uint) {
	time.Sleep(10 * time.Second)

	var userSessions []models.UserSession
	if err := inits.DB.Where("session_id = ?", sessionID).Find(&userSessions).Error; err != nil {
		log.Printf("[ConvertSessionTsToMP4] Failed to load user sessions for session %d: %v\n", sessionID, err)
		return
	}

	vodFolder := filepath.Join("videos", fmt.Sprintf("%d", sessionID))
	if err := os.MkdirAll(vodFolder, os.ModePerm); err != nil {
		log.Printf("[ConvertSessionTsToMP4] Failed to create VOD folder for session %d: %v\n", sessionID, err)
		return
	}

	for _, us := range userSessions {
		userID := us.UserID
		tsPath := filepath.Join("uploads", fmt.Sprintf("%d", sessionID), fmt.Sprintf("%d", userID), "dash", "stream.ts")
		outputPath := filepath.Join(vodFolder, fmt.Sprintf("%d.mp4", userID))

		tsPathAbs, err := filepath.Abs(tsPath)
		if err != nil {
			log.Printf("[ConvertSessionTsToMP4] Failed to get absolute TS path for user %d: %v\n", userID, err)
			continue
		}
		outputPathAbs, err := filepath.Abs(outputPath)
		if err != nil {
			log.Printf("[ConvertSessionTsToMP4] Failed to get absolute output path for user %d: %v\n", userID, err)
			continue
		}

		go func(userID uint, input, output string) {
			log.Printf("[ConvertSessionTsToMP4] Starting MP4 conversion for user %d\n", userID)
			cmd := exec.Command("ffmpeg", "-y", "-i", input, "-c:v", "libx264", "-c:a", "aac", "-strict", "experimental", output)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				log.Printf("[ConvertSessionTsToMP4] MP4 conversion failed for user %d: %v\n", userID, err)
			} else {
				log.Printf("[ConvertSessionTsToMP4] MP4 conversion complete for user %d\n", userID)
			}
		}(userID, tsPathAbs, outputPathAbs)
	}
}

package controllers

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
)

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

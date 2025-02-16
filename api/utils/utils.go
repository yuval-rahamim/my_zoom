package utils

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

// RunCommand executes a command in the shell and handles errors
func RunCommand(command string) error {
	// Ensure cross-platform file paths
	if runtime.GOOS == "windows" {
		command = strings.ReplaceAll(command, "\\", "/") // Convert Windows paths to Unix-style for FFmpeg
	}

	var cmd *exec.Cmd

	// Check OS and set the right shell
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command) // Use cmd.exe on Windows
	} else {
		cmd = exec.Command("bash", "-c", command) // Use bash on Linux/Mac
	}

	// Run the command and capture the output
	output, err := cmd.CombinedOutput()
	log.Printf("Command: %s\nOutput: %s\n", command, string(output)) // Log command and output

	// If the command failed, log the error and return it
	if err != nil {
		log.Printf("Command failed: %v\nstderr: %s\n", err, string(output))
		return fmt.Errorf("Command failed: %v, stderr: %s", err, string(output))
	}
	return nil
}

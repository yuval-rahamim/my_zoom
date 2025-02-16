package utils

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
)

// RunCommand executes a command in the shell and handles errors
func RunCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)

	output, err := cmd.CombinedOutput() // Capture stdout and stderr
	if err != nil {
		log.Printf("Command failed: %v\nstderr: %s\n", err, string(output))
		return fmt.Errorf("Command failed: %v, stderr: %s", err, string(output))
	}
	return nil
}

// RunCommand executes a command in the shell and handles errors
func WindowesRunCommand(command string) error {
	var cmd *exec.Cmd

	// Check OS and set the right shell
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command) // Use cmd.exe on Windows
	} else {
		cmd = exec.Command("bash", "-c", command) // Use bash on Linux/Mac
	}

	output, err := cmd.CombinedOutput() // Capture stdout and stderr
	if err != nil {
		log.Printf("Command failed: %v\nstderr: %s\n", err, string(output))
		return fmt.Errorf("Command failed: %v, stderr: %s", err, string(output))
	}
	return nil
}

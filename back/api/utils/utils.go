package utils

import (
	"fmt"
	"log"
	"os/exec"
)

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

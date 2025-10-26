package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-wyvern/x-agent/pkg/container"
)

func main() {
	manager, err := container.NewManager("xagent:latest")
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	manager.StartCleanupScheduler(30*time.Minute, 2*time.Hour)

	sessionID := "example-session-001"
	workspacePath := "/tmp/workspace"

	c, err := manager.CreateContainer(sessionID, workspacePath)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	fmt.Printf("Created container: %s\n", c.ContainerName)

	status, err := manager.CheckContainerStatus(sessionID)
	if err != nil {
		log.Fatalf("Failed to check status: %v", err)
	}
	fmt.Printf("Container status: %s\n", status)

	message := "List the files in the current directory"
	reader, err := manager.Prompt(sessionID, message)
	if err != nil {
		log.Fatalf("Failed to execute claude: %v", err)
	}
	defer reader.Close()
	
	output := make([]byte, 1024*64)
	n, _ := reader.Read(output)
	fmt.Printf("Claude output:\n%s\n", string(output[:n]))

	logs, err := manager.GetContainerLogs(sessionID, "50")
	if err != nil {
		log.Fatalf("Failed to get logs: %v", err)
	}
	fmt.Printf("Container logs:\n%s\n", logs)

	err = manager.StopContainer(sessionID)
	if err != nil {
		log.Fatalf("Failed to stop container: %v", err)
	}
	fmt.Println("Container stopped successfully")
}

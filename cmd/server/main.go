package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-wyvern/x-agent/internal/api"
	"github.com/go-wyvern/x-agent/internal/session"
	"github.com/go-wyvern/x-agent/pkg/container"
	"github.com/go-wyvern/x-agent/pkg/storage"
)

func main() {
	store := storage.NewRedisSessionStore("localhost:6379", 24*time.Hour)

	sessionManager := session.NewSessionManager(store, 24*time.Hour)

	sessionManager.StartCleanupScheduler(1 * time.Hour)

	containerManager, err := container.NewManager("ghcr.io/anthropics/claude-code:latest")
	if err != nil {
		log.Fatalf("Failed to create container manager: %v", err)
	}
	defer containerManager.Close()

	containerManager.StartCleanupScheduler(1*time.Hour, 24*time.Hour)

	chatHandler := api.NewChatHandler(sessionManager, containerManager)

	r := gin.Default()

	r.POST("/v1/chat/completions", chatHandler.HandleChatCompletion)
	r.GET("/v1/sessions/:id", chatHandler.HandleGetSession)
	r.DELETE("/v1/sessions/:id", chatHandler.HandleDeleteSession)

	log.Println("Starting x-agent server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

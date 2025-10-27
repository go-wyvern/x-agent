package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-wyvern/x-agent/internal/api"
	"github.com/go-wyvern/x-agent/internal/session"
	"github.com/go-wyvern/x-agent/internal/wechat"
	"github.com/go-wyvern/x-agent/pkg/container"
	"github.com/go-wyvern/x-agent/pkg/storage"
)

func main() {
	store := storage.NewRedisSessionStore("localhost:6379", 24*time.Hour)

	sessionManager := session.NewSessionManager(store, 24*time.Hour)

	sessionManager.StartCleanupScheduler(1 * time.Hour)

	containerConfig := &container.Config{
		ImageTag:              os.Getenv("IMAGE_TAG"),
		AnthropicAPIKey:       os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicBaseURL:      os.Getenv("ANTHROPIC_BASE_URL"),
		AnthropicAuthToken:    os.Getenv("ANTHROPIC_AUTH_TOKEN"),
		AnthropicModel:        os.Getenv("ANTHROPIC_MODEL"),
		AnthropicDefaultModel: os.Getenv("ANTHROPIC_DEFAULT_SONNET_MODEL"),
	}

	if containerConfig.ImageTag == "" {
		containerConfig.ImageTag = "ghcr.io/anthropics/claude-code:latest"
	}

	containerManager, err := container.NewManager(containerConfig)
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

	wechatConfig := loadWechatConfig()
	if wechatConfig != nil {
		wechatHandler, err := wechat.NewHandler(wechatConfig, sessionManager, containerManager)
		if err != nil {
			log.Printf("Failed to create wechat handler: %v", err)
		} else {
			r.POST("/api/wechat/webhook", wechatHandler.ReceiveWebhook)
			log.Println("WeChat Work Smart Robot integration enabled")
		}
	}

	log.Println("Starting x-agent server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func loadWechatConfig() *wechat.Config {
	webhookKey := os.Getenv("WECHAT_WEBHOOK_KEY")
	if webhookKey == "" {
		return nil
	}

	return &wechat.Config{
		WebhookKey: webhookKey,
	}
}

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
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost:6379"
	}

	store := storage.NewRedisSessionStore(redisHost, 24*time.Hour)

	sessionManager := session.NewSessionManager(store, 24*time.Hour)

	sessionManager.StartCleanupScheduler(1 * time.Hour)

	containerConfig := &container.Config{
		ImageTag:                    os.Getenv("IMAGE_TAG"),
		AnthropicBaseURL:            os.Getenv("ANTHROPIC_BASE_URL"),
		AnthropicAuthToken:          os.Getenv("ANTHROPIC_AUTH_TOKEN"),
		AnthropicModel:              os.Getenv("ANTHROPIC_MODEL"),
		AnthropicDefaultSonnetModel: os.Getenv("ANTHROPIC_DEFAULT_SONNET_MODEL"),
		AnthropicDefaultHaikuModel:  os.Getenv("ANTHROPIC_DEFAULT_HAIKU_MODEL"),
		AnthropicSmallFastModel:     os.Getenv("ANTHROPIC_SMALL_FAST_MODEL"),
	}

	if containerConfig.ImageTag == "" {
		containerConfig.ImageTag = "x-agent:latest"
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

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	log.Printf("Starting x-agent server on :%s", serverPort)
	if err := r.Run(":" + serverPort); err != nil {
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

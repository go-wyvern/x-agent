package main

import (
	"log"
	"os"
	"strconv"
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

	wechatConfig := loadWechatConfig()
	if wechatConfig != nil {
		wechatHandler, err := wechat.NewHandler(wechatConfig, sessionManager, containerManager)
		if err != nil {
			log.Printf("Failed to create wechat handler: %v", err)
		} else {
			r.GET("/api/wechat/callback", wechatHandler.VerifyURL)
			r.POST("/api/wechat/callback", wechatHandler.ReceiveMessage)
			log.Println("WeChat Work integration enabled")
		}
	}

	log.Println("Starting x-agent server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func loadWechatConfig() *wechat.Config {
	corpID := os.Getenv("WECHAT_CORP_ID")
	if corpID == "" {
		return nil
	}

	agentIDStr := os.Getenv("WECHAT_AGENT_ID")
	if agentIDStr == "" {
		return nil
	}

	agentID, err := strconv.Atoi(agentIDStr)
	if err != nil {
		log.Printf("Invalid WECHAT_AGENT_ID: %v", err)
		return nil
	}

	secret := os.Getenv("WECHAT_SECRET")
	token := os.Getenv("WECHAT_TOKEN")
	encodingAESKey := os.Getenv("WECHAT_ENCODING_AES_KEY")

	if secret == "" || token == "" || encodingAESKey == "" {
		return nil
	}

	return &wechat.Config{
		CorpID:         corpID,
		AgentID:        agentID,
		Secret:         secret,
		Token:          token,
		EncodingAESKey: encodingAESKey,
	}
}

package wechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/go-wyvern/x-agent/internal/session"
	"github.com/go-wyvern/x-agent/pkg/container"
)

type Handler struct {
	config           *Config
	crypto           *Crypto
	sessionManager   *session.SessionManager
	containerManager *container.Manager
	userSessionStore *UserSessionStore
}

func NewHandler(
	config *Config,
	sessionManager *session.SessionManager,
	containerManager *container.Manager,
) (*Handler, error) {
	crypto, err := NewCrypto(config.WebhookKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto: %w", err)
	}

	return &Handler{
		config:           config,
		crypto:           crypto,
		sessionManager:   sessionManager,
		containerManager: containerManager,
		userSessionStore: NewUserSessionStore(),
	}, nil
}

func (h *Handler) ReceiveWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var msg WebhookMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	go h.processWebhookMessage(&msg)

	c.JSON(http.StatusOK, gin.H{"code": 0})
}

func (h *Handler) processWebhookMessage(msg *WebhookMessage) {
	if msg.MsgType != "text" {
		log.Printf("Ignoring non-text message type: %s", msg.MsgType)
		return
	}

	chatID := msg.ChatID
	chatType := msg.ChatType
	userID := msg.From.UserID
	content := msg.Text.Content

	isGroupChat := chatType == "group"

	if isGroupChat {
		if !h.isMentioned(content) {
			log.Printf("Message in group %s does not mention bot, ignoring", chatID)
			return
		}
		content = h.extractContent(content)
	}

	if strings.HasPrefix(content, "/") {
		h.handleCommand(msg, content, isGroupChat)
		return
	}

	h.sendWebhookMessage(msg.WebhookURL, "正在思考中...")

	sessionKey := userID
	if isGroupChat {
		sessionKey = chatID
	}

	sessionID := h.userSessionStore.GetSession(sessionKey)
	if sessionID == "" {
		sess, err := h.sessionManager.CreateSession(sessionKey)
		if err != nil {
			log.Printf("Failed to create session for %s: %v", sessionKey, err)
			h.sendWebhookMessage(msg.WebhookURL, "抱歉，创建会话失败，请稍后重试")
			return
		}
		sessionID = sess.ID
		h.userSessionStore.SetSession(sessionKey, sessionID)

		_, err = h.containerManager.CreateContainer(sessionID, "/workspace")
		if err != nil {
			log.Printf("Failed to create container for session %s: %v", sessionID, err)
			h.sendWebhookMessage(msg.WebhookURL, "抱歉，创建容器失败，请稍后重试")
			return
		}
	}

	messageContent := content
	if isGroupChat {
		messageContent = fmt.Sprintf("[%s]: %s", msg.From.Name, content)
	}

	if err := h.sessionManager.AddMessage(sessionID, "user", messageContent); err != nil {
		log.Printf("Failed to save user message: %v", err)
	}

	container, err := h.containerManager.GetContainer(sessionID)
	if err != nil {
		log.Printf("Failed to get container: %v", err)
		h.sendWebhookMessage(msg.WebhookURL, "抱歉，容器未找到，请稍后重试")
		return
	}

	responseStream, err := container.Prompt(content)
	if err != nil {
		log.Printf("Failed to execute prompt: %v", err)
		h.sendWebhookMessage(msg.WebhookURL, "抱歉，处理消息时出错了，请稍后重试")
		return
	}
	defer responseStream.Close()

	responseBytes, err := io.ReadAll(responseStream)
	if err != nil {
		log.Printf("Failed to read response: %v", err)
		h.sendWebhookMessage(msg.WebhookURL, "抱歉，读取响应失败，请稍后重试")
		return
	}

	assistantResponse := string(responseBytes)

	if err := h.sessionManager.AddMessage(sessionID, "assistant", assistantResponse); err != nil {
		log.Printf("Failed to save assistant message: %v", err)
	}

	formattedResponse := FormatMarkdown(assistantResponse)
	h.sendWebhookMessage(msg.WebhookURL, formattedResponse)
}

func (h *Handler) isMentioned(content string) bool {
	mentions := []string{
		"@x-agent",
		"@X-Agent",
		"@x-Agent",
	}

	for _, mention := range mentions {
		if strings.Contains(content, mention) {
			return true
		}
	}

	return false
}

func (h *Handler) extractContent(content string) string {
	content = strings.ReplaceAll(content, "@x-agent", "")
	content = strings.ReplaceAll(content, "@X-Agent", "")
	content = strings.ReplaceAll(content, "@x-Agent", "")

	content = strings.TrimSpace(content)

	return content
}

func (h *Handler) handleCommand(msg *WebhookMessage, command string, isGroupChat bool) {
	command = strings.TrimSpace(command)

	sessionKey := msg.From.UserID
	if isGroupChat {
		sessionKey = msg.ChatID
	}

	switch {
	case command == "/help":
		helpText := `可用命令:
/help - 显示帮助
/new - 开始新对话
/reset - 重置当前会话
/status - 查看会话状态`
		h.sendWebhookMessage(msg.WebhookURL, helpText)

	case command == "/new":
		oldSessionID := h.userSessionStore.GetSession(sessionKey)
		if oldSessionID != "" {
			if err := h.sessionManager.DeleteSession(oldSessionID); err != nil {
				log.Printf("Failed to delete old session %s: %v", oldSessionID, err)
			}
		}

		sess, err := h.sessionManager.CreateSession(sessionKey)
		if err != nil {
			h.sendWebhookMessage(msg.WebhookURL, "创建新会话失败，请稍后重试")
			return
		}

		_, err = h.containerManager.CreateContainer(sess.ID, "/workspace")
		if err != nil {
			if err := h.sessionManager.DeleteSession(sess.ID); err != nil {
				log.Printf("Failed to cleanup session %s: %v", sess.ID, err)
			}
			h.sendWebhookMessage(msg.WebhookURL, "创建容器失败，请稍后重试")
			return
		}

		h.userSessionStore.SetSession(sessionKey, sess.ID)
		h.sendWebhookMessage(msg.WebhookURL, fmt.Sprintf("已创建新会话: %s", sess.ID[:8]))

	case command == "/reset":
		sessionID := h.userSessionStore.GetSession(sessionKey)
		if sessionID == "" {
			h.sendWebhookMessage(msg.WebhookURL, "当前无活跃会话")
			return
		}

		if err := h.sessionManager.DeleteSession(sessionID); err != nil {
			log.Printf("Failed to delete session %s: %v", sessionID, err)
		}
		h.userSessionStore.ClearSession(sessionKey)
		h.sendWebhookMessage(msg.WebhookURL, "会话已重置")

	case command == "/status":
		sessionID := h.userSessionStore.GetSession(sessionKey)
		if sessionID == "" {
			h.sendWebhookMessage(msg.WebhookURL, "当前无活跃会话")
		} else {
			sess, err := h.sessionManager.GetSession(sessionID)
			if err != nil {
				h.sendWebhookMessage(msg.WebhookURL, "无法获取会话信息")
				return
			}
			statusText := fmt.Sprintf("会话 ID: %s\n消息数: %d\n创建时间: %s",
				sessionID[:8],
				len(sess.Messages),
				sess.CreatedAt.Format("2006-01-02 15:04:05"),
			)
			h.sendWebhookMessage(msg.WebhookURL, statusText)
		}

	default:
		h.sendWebhookMessage(msg.WebhookURL, "未知命令，输入 /help 查看帮助")
	}
}

func (h *Handler) sendWebhookMessage(webhookURL, content string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook url is empty")
	}

	response := WebhookResponse{
		MsgType: "text",
		Text: WebhookTextContent{
			Content: content,
		},
	}

	bodyBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to send webhook message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

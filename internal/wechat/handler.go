package wechat

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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
	tokenManager     *TokenManager
	sessionManager   *session.SessionManager
	containerManager *container.Manager
	userSessionStore *UserSessionStore
}

func NewHandler(
	config *Config,
	sessionManager *session.SessionManager,
	containerManager *container.Manager,
) (*Handler, error) {
	crypto, err := NewCrypto(config.Token, config.EncodingAESKey, config.CorpID)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto: %w", err)
	}

	return &Handler{
		config:           config,
		crypto:           crypto,
		tokenManager:     NewTokenManager(config),
		sessionManager:   sessionManager,
		containerManager: containerManager,
		userSessionStore: NewUserSessionStore(),
	}, nil
}

func (h *Handler) VerifyURL(c *gin.Context) {
	msgSignature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echostr := c.Query("echostr")

	decrypted, err := h.crypto.DecryptMsg(msgSignature, timestamp, nonce, echostr)
	if err != nil {
		log.Printf("Failed to decrypt echostr: %v", err)
		c.String(http.StatusBadRequest, "verification failed")
		return
	}

	c.String(http.StatusOK, decrypted)
}

func (h *Handler) ReceiveMessage(c *gin.Context) {
	msgSignature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		c.String(http.StatusBadRequest, "failed to read body")
		return
	}

	var encryptedMsg struct {
		Encrypt string `xml:"Encrypt"`
	}
	if err := xml.Unmarshal(body, &encryptedMsg); err != nil {
		log.Printf("Failed to unmarshal request: %v", err)
		c.String(http.StatusBadRequest, "invalid xml")
		return
	}

	decrypted, err := h.crypto.DecryptMsg(msgSignature, timestamp, nonce, encryptedMsg.Encrypt)
	if err != nil {
		log.Printf("Failed to decrypt message: %v", err)
		c.String(http.StatusBadRequest, "decryption failed")
		return
	}

	var msg Message
	if err := xml.Unmarshal([]byte(decrypted), &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		c.String(http.StatusBadRequest, "invalid message format")
		return
	}

	go h.processMessage(&msg)

	c.String(http.StatusOK, "success")
}

func (h *Handler) processMessage(msg *Message) {
	userID := msg.FromUserName
	content := msg.Content

	if strings.HasPrefix(content, "/") {
		h.handleCommand(userID, content)
		return
	}

	h.sendTypingIndicator(userID)

	sessionID := h.userSessionStore.GetSession(userID)
	if sessionID == "" {
		sess, err := h.sessionManager.CreateSession(userID)
		if err != nil {
			log.Printf("Failed to create session for user %s: %v", userID, err)
			h.sendMessage(userID, "抱歉，创建会话失败，请稍后重试")
			return
		}
		sessionID = sess.ID
		h.userSessionStore.SetSession(userID, sessionID)

		_, err = h.containerManager.CreateContainer(sessionID, "/workspace")
		if err != nil {
			log.Printf("Failed to create container for session %s: %v", sessionID, err)
			h.sendMessage(userID, "抱歉，创建容器失败，请稍后重试")
			return
		}
	}

	if err := h.sessionManager.AddMessage(sessionID, "user", content); err != nil {
		log.Printf("Failed to save user message: %v", err)
	}

	responseStream, err := h.containerManager.Prompt(sessionID, content)
	if err != nil {
		log.Printf("Failed to execute prompt: %v", err)
		h.sendMessage(userID, "抱歉，处理消息时出错了，请稍后重试")
		return
	}
	defer responseStream.Close()

	responseBytes, err := io.ReadAll(responseStream)
	if err != nil {
		log.Printf("Failed to read response: %v", err)
		h.sendMessage(userID, "抱歉，读取响应失败，请稍后重试")
		return
	}

	assistantResponse := string(responseBytes)

	if err := h.sessionManager.AddMessage(sessionID, "assistant", assistantResponse); err != nil {
		log.Printf("Failed to save assistant message: %v", err)
	}

	formattedResponse := FormatMarkdown(assistantResponse)
	h.sendMessage(userID, formattedResponse)
}

func (h *Handler) handleCommand(userID, command string) {
	command = strings.TrimSpace(command)

	switch {
	case command == "/help":
		helpText := `可用命令:
/help - 显示帮助
/new - 开始新对话
/reset - 重置当前会话
/status - 查看会话状态`
		h.sendMessage(userID, helpText)

	case command == "/new":
		oldSessionID := h.userSessionStore.GetSession(userID)
		if oldSessionID != "" {
			h.sessionManager.DeleteSession(oldSessionID)
		}

		sess, err := h.sessionManager.CreateSession(userID)
		if err != nil {
			h.sendMessage(userID, "创建新会话失败，请稍后重试")
			return
		}

		_, err = h.containerManager.CreateContainer(sess.ID, "/workspace")
		if err != nil {
			h.sendMessage(userID, "创建容器失败，请稍后重试")
			return
		}

		h.userSessionStore.SetSession(userID, sess.ID)
		h.sendMessage(userID, fmt.Sprintf("已创建新会话: %s", sess.ID[:8]))

	case command == "/reset":
		sessionID := h.userSessionStore.GetSession(userID)
		if sessionID == "" {
			h.sendMessage(userID, "当前无活跃会话")
			return
		}

		h.sessionManager.DeleteSession(sessionID)
		h.userSessionStore.ClearSession(userID)
		h.sendMessage(userID, "会话已重置")

	case command == "/status":
		sessionID := h.userSessionStore.GetSession(userID)
		if sessionID == "" {
			h.sendMessage(userID, "当前无活跃会话")
		} else {
			sess, err := h.sessionManager.GetSession(sessionID)
			if err != nil {
				h.sendMessage(userID, "无法获取会话信息")
				return
			}
			statusText := fmt.Sprintf("会话 ID: %s\n消息数: %d\n创建时间: %s",
				sessionID[:8],
				len(sess.Messages),
				sess.CreatedAt.Format("2006-01-02 15:04:05"),
			)
			h.sendMessage(userID, statusText)
		}

	default:
		h.sendMessage(userID, "未知命令，输入 /help 查看帮助")
	}
}

func (h *Handler) sendTypingIndicator(userID string) {
	h.sendMessage(userID, "正在思考中...")
}

func (h *Handler) sendMessage(userID, content string) error {
	accessToken, err := h.tokenManager.GetAccessToken()
	if err != nil {
		log.Printf("Failed to get access token: %v", err)
		return err
	}

	msg := TextResponse{
		ToUser:  userID,
		MsgType: "text",
		AgentID: h.config.AgentID,
		Text: TextContent{
			Content: content,
		},
	}

	url := fmt.Sprintf(
		"https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s",
		accessToken,
	)

	bodyBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	var result APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wechat api error: %s (code: %d)", result.ErrMsg, result.ErrCode)
	}

	return nil
}

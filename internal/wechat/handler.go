package wechat

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

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
	streamStore      *StreamStore
}

func NewHandler(
	config *Config,
	sessionManager *session.SessionManager,
	containerManager *container.Manager,
) (*Handler, error) {
	crypto, err := NewCrypto(config.Token, config.EncodingAESKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto: %w", err)
	}

	return &Handler{
		config:           config,
		crypto:           crypto,
		sessionManager:   sessionManager,
		containerManager: containerManager,
		userSessionStore: NewUserSessionStore(),
		streamStore:      NewStreamStore(),
	}, nil
}

func (h *Handler) HandleCallback(c *gin.Context) {
	if c.Request.Method == "GET" {
		h.verifyURL(c)
		return
	}

	h.handleMessage(c)
}

func (h *Handler) verifyURL(c *gin.Context) {
	msgSignature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	echoStr := c.Query("echostr")

	log.Printf("Verifying URL: msg_signature=%s, timestamp=%s, nonce=%s", msgSignature, timestamp, nonce)

	decrypted, err := h.crypto.VerifyURL(msgSignature, timestamp, nonce, echoStr)
	if err != nil {
		log.Printf("Failed to verify URL: %v", err)
		c.String(http.StatusBadRequest, "verify fail")
		return
	}

	log.Printf("URL verification successful: %s", decrypted)
	c.String(http.StatusOK, decrypted)
}

func (h *Handler) handleMessage(c *gin.Context) {
	msgSignature := c.Query("msg_signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")

	if msgSignature == "" || timestamp == "" || nonce == "" {
		log.Printf("Missing required parameters")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing parameters"})
		return
	}

	log.Printf("Received message: msg_signature=%s, timestamp=%s, nonce=%s", msgSignature, timestamp, nonce)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var encReq EncryptedRequest
	if err := json.Unmarshal(body, &encReq); err != nil {
		log.Printf("Failed to unmarshal encrypted request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	decrypted, err := h.crypto.DecryptMsg(encReq.Encrypt, msgSignature, timestamp, nonce, "")
	if err != nil {
		log.Printf("Failed to decrypt message: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "decryption failed"})
		return
	}

	log.Printf("Decrypted message: %s", decrypted)

	var msg IncomingMessage
	if err := json.Unmarshal([]byte(decrypted), &msg); err != nil {
		log.Printf("Failed to unmarshal decrypted message: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message format"})
		return
	}

	response, err := h.processMessage(&msg, nonce, timestamp)
	if err != nil {
		log.Printf("Failed to process message: %v", err)
		c.String(http.StatusOK, "success")
		return
	}

	if response != "" {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, response)
	} else {
		c.String(http.StatusOK, "success")
	}
}

func (h *Handler) processMessage(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.MsgType == "" {
		log.Printf("Unknown message type: %+v", msg)
		return "", nil
	}

	switch msg.MsgType {
	case "text":
		return h.handleTextMessage(msg, nonce, timestamp)
	case "stream":
		return h.handleStreamMessage(msg, nonce, timestamp)
	case "image":
		return h.handleImageMessage(msg, nonce, timestamp)
	case "mixed":
		log.Printf("Mixed message type not yet supported")
		return "", nil
	case "event":
		log.Printf("Event message: %+v", msg.Event)
		return "", nil
	default:
		log.Printf("Unsupported message type: %s", msg.MsgType)
		return "", nil
	}
}

func (h *Handler) handleTextMessage(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Text == nil {
		return "", fmt.Errorf("text message has no content")
	}

	content := msg.Text.Content
	log.Printf("Received text message: %s", content)

	streamID := generateStreamID()
	botID := msg.ToUserName
	if botID == "" {
		botID = "bot_default"
	}

	sessionID := h.userSessionStore.GetSession(botID)
	if sessionID == "" {
		sess, err := h.sessionManager.CreateSession(botID)
		if err != nil {
			log.Printf("Failed to create session for %s: %v", botID, err)
			return h.createErrorResponse(streamID, "创建会话失败", nonce, timestamp)
		}
		sessionID = sess.ID
		h.userSessionStore.SetSession(botID, sessionID)

		_, err = h.containerManager.CreateContainer(sessionID, "/workspace")
		if err != nil {
			log.Printf("Failed to create container for session %s: %v", sessionID, err)
			return h.createErrorResponse(streamID, "创建容器失败", nonce, timestamp)
		}
	}

	h.streamStore.SetStreamData(streamID, &StreamData{
		SessionID:      sessionID,
		UserID:         botID,
		Question:       content,
		Step:           0,
		MaxSteps:       10,
		ResponseChunks: []string{},
	})

	if err := h.sessionManager.AddMessage(sessionID, "user", content); err != nil {
		log.Printf("Failed to save user message: %v", err)
	}

	container, err := h.containerManager.GetContainer(sessionID)
	if err != nil {
		log.Printf("Failed to get container: %v", err)
		return h.createErrorResponse(streamID, "容器未找到", nonce, timestamp)
	}

	go func() {
		responseStream, err := container.Prompt(content)
		if err != nil {
			log.Printf("Failed to execute prompt: %v", err)
			return
		}
		defer responseStream.Close()

		responseBytes, err := io.ReadAll(responseStream)
		if err != nil {
			log.Printf("Failed to read response: %v", err)
			return
		}

		assistantResponse := string(responseBytes)
		if err := h.sessionManager.AddMessage(sessionID, "assistant", assistantResponse); err != nil {
			log.Printf("Failed to save assistant message: %v", err)
		}

		data := h.streamStore.GetStreamData(streamID)
		if data != nil {
			data.ResponseChunks = append(data.ResponseChunks, assistantResponse)
			h.streamStore.SetStreamData(streamID, data)
		}
	}()

	answer := fmt.Sprintf("收到问题：%s\n处理步骤 0: 已完成\n", content)
	return h.createTextStreamResponse(streamID, answer, false, nonce, timestamp)
}

func (h *Handler) handleStreamMessage(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Stream == nil {
		return "", fmt.Errorf("stream message has no stream data")
	}

	streamID := msg.Stream.ID
	log.Printf("Received stream request for ID: %s", streamID)

	data := h.streamStore.GetStreamData(streamID)
	if data == nil {
		log.Printf("Stream data not found for ID: %s", streamID)
		return h.createTextStreamResponse(streamID, "任务不存在或已过期", true, nonce, timestamp)
	}

	// Check if response chunks are available
	var answer string
	var finish bool

	if len(data.ResponseChunks) > 0 {
		// Response is ready, return it
		answer = data.ResponseChunks[0]
		finish = true
		log.Printf("Returning response for stream %s: %d bytes", streamID, len(answer))
	} else {
		// Response not ready yet, increment step and return progress
		data.Step++
		h.streamStore.SetStreamData(streamID, data)

		answer = fmt.Sprintf("收到问题：%s\n", data.Question)
		for i := 0; i < data.Step; i++ {
			answer += fmt.Sprintf("处理步骤 %d: 已完成\n", i)
		}

		// Check if we've reached max steps without a response
		finish = data.Step >= data.MaxSteps
	}

	return h.createTextStreamResponse(streamID, answer, finish, nonce, timestamp)
}

func (h *Handler) handleImageMessage(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Image == nil {
		return "", fmt.Errorf("image message has no image data")
	}

	log.Printf("Received image message: %s", msg.Image.URL)

	imageData, err := h.downloadAndDecryptImage(msg.Image.URL)
	if err != nil {
		log.Printf("Failed to process image: %v", err)
		streamID := generateStreamID()
		return h.createErrorResponse(streamID, "图片处理失败", nonce, timestamp)
	}

	streamID := generateStreamID()
	return h.createImageStreamResponse(streamID, imageData, true, nonce, timestamp)
}

func (h *Handler) downloadAndDecryptImage(imageURL string) ([]byte, error) {
	resp, err := http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image download returned status %d", resp.StatusCode)
	}

	encryptedData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	log.Printf("Downloaded encrypted image, size: %d bytes", len(encryptedData))

	decryptedData, err := h.decryptImageData(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt image: %w", err)
	}

	log.Printf("Image decrypted successfully, size: %d bytes", len(decryptedData))
	return decryptedData, nil
}

func (h *Handler) decryptImageData(encryptedData []byte) ([]byte, error) {
	aesKey := h.crypto.aesKey
	iv := aesKey[:16]

	return decryptAESCBC(encryptedData, aesKey, iv)
}

func (h *Handler) createTextStreamResponse(streamID, content string, finish bool, nonce, timestamp string) (string, error) {
	stream := StreamResponse{
		MsgType: "stream",
		Stream: StreamResponseData{
			ID:      streamID,
			Finish:  finish,
			Content: content,
		},
	}

	streamJSON, err := json.Marshal(stream)
	if err != nil {
		return "", fmt.Errorf("failed to marshal stream response: %w", err)
	}

	log.Printf("Sending stream response: stream_id=%s, finish=%v", streamID, finish)
	return h.crypto.EncryptMsg(string(streamJSON), nonce, timestamp)
}

func (h *Handler) createImageStreamResponse(streamID string, imageData []byte, finish bool, nonce, timestamp string) (string, error) {
	imageMD5 := md5.Sum(imageData)
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	stream := StreamResponse{
		MsgType: "stream",
		Stream: StreamResponseData{
			ID:     streamID,
			Finish: finish,
			MsgItem: []StreamMsgItem{
				{
					MsgType: "image",
					Image: &ImageMsgItem{
						Base64: imageBase64,
						MD5:    hex.EncodeToString(imageMD5[:]),
					},
				},
			},
		},
	}

	streamJSON, err := json.Marshal(stream)
	if err != nil {
		return "", fmt.Errorf("failed to marshal image stream response: %w", err)
	}

	log.Printf("Sending image stream response: stream_id=%s", streamID)
	return h.crypto.EncryptMsg(string(streamJSON), nonce, timestamp)
}

func (h *Handler) createErrorResponse(streamID, errorMsg string, nonce, timestamp string) (string, error) {
	return h.createTextStreamResponse(streamID, errorMsg, true, nonce, timestamp)
}

func generateStreamID() string {
	return fmt.Sprintf("stream_%d", time.Now().UnixNano())
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
	resp, err := client.Post(webhookURL, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("failed to send webhook message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

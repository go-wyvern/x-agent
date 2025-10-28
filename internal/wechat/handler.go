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
	case "event":
		return h.handleEvent(msg, nonce, timestamp)
	case "text":
		return h.handleText(msg, nonce, timestamp)
	case "stream":
		return h.handleStream(msg, nonce, timestamp)
	case "image":
		return h.handleImage(msg, nonce, timestamp)
	case "mixed":
		return h.handleMix(msg, nonce, timestamp)
	default:
		log.Printf("Unsupported message type: %s", msg.MsgType)
		return "", nil
	}
}

func (h *Handler) handleEvent(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Event == nil {
		return "", fmt.Errorf("event message has no event data")
	}

	log.Printf("Received event: %s", msg.Event.EventType)

	switch msg.Event.EventType {
	case "enter_chat":
		return h.handleEnterChatEvent(msg, nonce, timestamp)
	case "template_card_event":
		return h.handleTemplateCardEvent(msg, nonce, timestamp)
	default:
		log.Printf("Unsupported event type: %s", msg.Event.EventType)
		return "", nil
	}
}

func (h *Handler) handleEnterChatEvent(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	response := TextResponse{
		MsgType: "text",
		Text: &TextMsg{
			Content: "您好,有什么可以帮你的?\n",
		},
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal text response: %w", err)
	}

	log.Printf("Sending enter_chat response")
	return h.crypto.EncryptMsg(string(responseJSON), nonce, timestamp)
}

func (h *Handler) handleTemplateCardEvent(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Event.TemplateCardEvent == nil {
		return "", fmt.Errorf("template card event has no event data")
	}

	log.Printf("Template card event: card_type=%s, event_key=%s, task_id=%s",
		msg.Event.TemplateCardEvent.CardType,
		msg.Event.TemplateCardEvent.EventKey,
		msg.Event.TemplateCardEvent.TaskID)

	return "", nil
}

func (h *Handler) handleText(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Text == nil {
		return "", fmt.Errorf("text message has no content")
	}

	content := msg.Text.Content
	log.Printf("Received text message: %s", content)

	streamID := generateStreamID()
	userID := msg.From.UserID
	if userID == "" {
		userID = "user_default"
	}

	sessionKey := h.getSessionKey(msg)
	sessionID := h.userSessionStore.GetSession(sessionKey)
	if sessionID == "" {
		sess, err := h.sessionManager.CreateSession(sessionKey)
		if err != nil {
			log.Printf("Failed to create session for %s: %v", sessionKey, err)
			return "", fmt.Errorf("failed to create session: %w", err)
		}
		sessionID = sess.ID
		h.userSessionStore.SetSession(sessionKey, sessionID)

		_, err = h.containerManager.CreateContainer(sessionID, "/workspace")
		if err != nil {
			log.Printf("Failed to create container for session %s: %v", sessionID, err)
			return "", fmt.Errorf("failed to create container: %w", err)
		}
	}

	h.streamStore.SetStreamData(streamID, &StreamData{
		SessionID: sessionID,
		UserID:    userID,
		Question:  content,
		Step:      0,
		MaxSteps:  10,
	})

	if err := h.sessionManager.AddMessage(sessionID, "user", content); err != nil {
		log.Printf("Failed to save user message: %v", err)
	}

	container, err := h.containerManager.GetContainer(sessionID)
	if err != nil {
		log.Printf("Failed to get container: %v", err)
		return "", fmt.Errorf("failed to get container: %w", err)
	}

	go func() {
		responseStream, err := container.Prompt(content)
		if err != nil {
			log.Printf("Failed to execute prompt: %v", err)
			data := h.streamStore.GetStreamData(streamID)
			if data != nil {
				data.Error = err.Error()
				data.Finish = true
				h.streamStore.SetStreamData(streamID, data)
			}
			return
		}

		h.processResponseStream(streamID, sessionID, responseStream)
	}()

	return h.createStreamIDResponse(streamID, nonce, timestamp)
}

func (h *Handler) handleStream(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Stream == nil {
		return "", fmt.Errorf("stream message has no stream data")
	}

	streamID := msg.Stream.ID
	log.Printf("Received stream refresh request for ID: %s", streamID)

	data := h.streamStore.GetStreamData(streamID)
	if data == nil {
		log.Printf("Stream data not found for ID: %s", streamID)
		return h.createTextStreamResponse(streamID, "任务不存在或已过期", true, nonce, timestamp)
	}

	if data.Error != "" {
		return h.createTextStreamResponse(streamID, fmt.Sprintf("处理出错: %s", data.Error), true, nonce, timestamp)
	}

	if data.Finish && data.Response != "" {
		return h.createTextStreamResponse(streamID, data.Response, true, nonce, timestamp)
	}

	if data.Response != "" {
		return h.createTextStreamResponse(streamID, data.Response, data.Finish, nonce, timestamp)
	}

	return h.createTextStreamResponse(streamID, "正在处理您的问题...", false, nonce, timestamp)
}

func (h *Handler) handleImage(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Image == nil {
		return "", fmt.Errorf("image message has no image data")
	}

	log.Printf("Received image message: %s", msg.Image.URL)

	streamID := generateStreamID()
	userID := msg.From.UserID
	if userID == "" {
		userID = "user_default"
	}

	sessionKey := h.getSessionKey(msg)
	sessionID := h.userSessionStore.GetSession(sessionKey)
	if sessionID == "" {
		sess, err := h.sessionManager.CreateSession(sessionKey)
		if err != nil {
			log.Printf("Failed to create session for %s: %v", sessionKey, err)
			return "", fmt.Errorf("failed to create session: %w", err)
		}
		sessionID = sess.ID
		h.userSessionStore.SetSession(sessionKey, sessionID)

		_, err = h.containerManager.CreateContainer(sessionID, "/workspace")
		if err != nil {
			log.Printf("Failed to create container for session %s: %v", sessionID, err)
			return "", fmt.Errorf("failed to create container: %w", err)
		}
	}

	h.streamStore.SetStreamData(streamID, &StreamData{
		SessionID: sessionID,
		UserID:    userID,
		Question:  "[Image uploaded]",
		Step:      0,
		MaxSteps:  10,
	})

	go func() {
		imageData, err := h.downloadAndDecryptImage(msg.Image.URL)
		if err != nil {
			log.Printf("Failed to process image: %v", err)
			data := h.streamStore.GetStreamData(streamID)
			if data != nil {
				data.Error = "图片处理失败"
				data.Finish = true
				h.streamStore.SetStreamData(streamID, data)
			}
			return
		}

		imageBase64 := base64.StdEncoding.EncodeToString(imageData)
		content := fmt.Sprintf("User uploaded an image (base64): %s", imageBase64)

		if err := h.sessionManager.AddMessage(sessionID, "user", content); err != nil {
			log.Printf("Failed to save user message: %v", err)
		}

		container, err := h.containerManager.GetContainer(sessionID)
		if err != nil {
			log.Printf("Failed to get container: %v", err)
			data := h.streamStore.GetStreamData(streamID)
			if data != nil {
				data.Error = "容器未找到"
				data.Finish = true
				h.streamStore.SetStreamData(streamID, data)
			}
			return
		}

		responseStream, err := container.Prompt("Please describe this image")
		if err != nil {
			log.Printf("Failed to execute prompt: %v", err)
			data := h.streamStore.GetStreamData(streamID)
			if data != nil {
				data.Error = err.Error()
				data.Finish = true
				h.streamStore.SetStreamData(streamID, data)
			}
			return
		}

		h.processResponseStream(streamID, sessionID, responseStream)
	}()

	return h.createStreamIDResponse(streamID, nonce, timestamp)
}

func (h *Handler) handleMix(msg *IncomingMessage, nonce, timestamp string) (string, error) {
	if msg.Mixed == nil {
		return "", fmt.Errorf("mixed message has no mixed data")
	}

	log.Printf("Received mixed message with %d items", len(msg.Mixed.MsgItem))

	streamID := generateStreamID()
	userID := msg.From.UserID
	if userID == "" {
		userID = "user_default"
	}

	sessionKey := h.getSessionKey(msg)
	sessionID := h.userSessionStore.GetSession(sessionKey)
	if sessionID == "" {
		sess, err := h.sessionManager.CreateSession(sessionKey)
		if err != nil {
			log.Printf("Failed to create session for %s: %v", sessionKey, err)
			return "", fmt.Errorf("failed to create session: %w", err)
		}
		sessionID = sess.ID
		h.userSessionStore.SetSession(sessionKey, sessionID)

		_, err = h.containerManager.CreateContainer(sessionID, "/workspace")
		if err != nil {
			log.Printf("Failed to create container for session %s: %v", sessionID, err)
			return "", fmt.Errorf("failed to create container: %w", err)
		}
	}

	h.streamStore.SetStreamData(streamID, &StreamData{
		SessionID: sessionID,
		UserID:    userID,
		Question:  "[Mixed content uploaded]",
		Step:      0,
		MaxSteps:  10,
	})

	go func() {
		var contentParts []string
		for _, item := range msg.Mixed.MsgItem {
			switch item.MsgType {
			case "text":
				if item.Text != nil {
					contentParts = append(contentParts, item.Text.Content)
				}
			case "image":
				if item.Image != nil {
					imageData, err := h.downloadAndDecryptImage(item.Image.URL)
					if err != nil {
						log.Printf("Failed to process image in mixed message: %v", err)
						continue
					}
					imageBase64 := base64.StdEncoding.EncodeToString(imageData)
					contentParts = append(contentParts, fmt.Sprintf("[Image: %s]", imageBase64))
				}
			}
		}

		content := strings.Join(contentParts, "\n")
		if err := h.sessionManager.AddMessage(sessionID, "user", content); err != nil {
			log.Printf("Failed to save user message: %v", err)
		}

		container, err := h.containerManager.GetContainer(sessionID)
		if err != nil {
			log.Printf("Failed to get container: %v", err)
			data := h.streamStore.GetStreamData(streamID)
			if data != nil {
				data.Error = "容器未找到"
				data.Finish = true
				h.streamStore.SetStreamData(streamID, data)
			}
			return
		}

		responseStream, err := container.Prompt(content)
		if err != nil {
			log.Printf("Failed to execute prompt: %v", err)
			data := h.streamStore.GetStreamData(streamID)
			if data != nil {
				data.Error = err.Error()
				data.Finish = true
				h.streamStore.SetStreamData(streamID, data)
			}
			return
		}

		h.processResponseStream(streamID, sessionID, responseStream)
	}()

	return h.createStreamIDResponse(streamID, nonce, timestamp)
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

func (h *Handler) createStreamIDResponse(streamID, nonce, timestamp string) (string, error) {
	stream := StreamResponse{
		MsgType: "stream",
		Stream: StreamResponseData{
			ID: streamID,
		},
	}

	streamJSON, err := json.Marshal(stream)
	if err != nil {
		return "", fmt.Errorf("failed to marshal stream ID response: %w", err)
	}

	log.Printf("Sending stream ID response: stream_id=%s", streamID)
	return h.crypto.EncryptMsg(string(streamJSON), nonce, timestamp)
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

func (h *Handler) processResponseStream(streamID string, sessionID string, responseStream io.ReadCloser) {
	defer responseStream.Close()

	buf := make([]byte, 4096)
	var fullResponse strings.Builder

	for {
		n, err := responseStream.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			fullResponse.WriteString(chunk)
			
			data := h.streamStore.GetStreamData(streamID)
			if data != nil {
				data.Response = fullResponse.String()
				data.Finish = false
				h.streamStore.SetStreamData(streamID, data)
			}
		}

		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading response: %v", err)
				data := h.streamStore.GetStreamData(streamID)
				if data != nil {
					data.Error = err.Error()
					data.Finish = true
					h.streamStore.SetStreamData(streamID, data)
				}
			}
			break
		}
	}

	assistantResponse := fullResponse.String()
	if err := h.sessionManager.AddMessage(sessionID, "assistant", assistantResponse); err != nil {
		log.Printf("Failed to save assistant message: %v", err)
	}

	data := h.streamStore.GetStreamData(streamID)
	if data != nil {
		data.Response = assistantResponse
		data.Finish = true
		h.streamStore.SetStreamData(streamID, data)
	}
}

func generateStreamID() string {
	return fmt.Sprintf("stream_%d", time.Now().UnixNano())
}

func (h *Handler) getSessionKey(msg *IncomingMessage) string {
	if msg.ChatID != "" {
		return fmt.Sprintf("chat:%s", msg.ChatID)
	}

	userID := msg.From.UserID
	if userID == "" {
		userID = "user_default"
	}
	return fmt.Sprintf("user:%s", userID)
}

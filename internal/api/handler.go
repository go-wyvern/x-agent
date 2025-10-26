package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-wyvern/x-agent/internal/models"
	"github.com/go-wyvern/x-agent/internal/session"
)

type ChatHandler struct {
	sessionManager *session.SessionManager
}

func NewChatHandler(sm *session.SessionManager) *ChatHandler {
	return &ChatHandler{
		sessionManager: sm,
	}
}

type ChatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []MessageInput  `json:"messages"`
	Stream   bool            `json:"stream"`
}

type MessageInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int           `json:"index"`
	Message      MessageOutput `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type MessageOutput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (h *ChatHandler) HandleChatCompletion(c *gin.Context) {
	var req ChatCompletionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "default"
	}

	var sess *models.Session
	var err error

	if sessionID == "" {
		sess, err = h.sessionManager.CreateSession(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			return
		}
		c.Header("X-Session-ID", sess.ID)
	} else {
		sess, err = h.sessionManager.GetSession(sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
	}

	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages cannot be empty"})
		return
	}

	userMessage := req.Messages[len(req.Messages)-1].Content
	if err := h.sessionManager.AddMessage(sess.ID, "user", userMessage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save message"})
		return
	}

	assistantResponse := "This is a placeholder response from x-agent"

	if err := h.sessionManager.AddMessage(sess.ID, "assistant", assistantResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save response"})
		return
	}

	c.JSON(http.StatusOK, ChatCompletionResponse{
		ID:      "chatcmpl-" + sess.ID,
		Object:  "chat.completion",
		Created: sess.UpdatedAt.Unix(),
		Model:   req.Model,
		Choices: []Choice{{
			Index: 0,
			Message: MessageOutput{
				Role:    "assistant",
				Content: assistantResponse,
			},
			FinishReason: "stop",
		}},
	})
}

func (h *ChatHandler) HandleGetSession(c *gin.Context) {
	sessionID := c.Param("id")

	sess, err := h.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, sess)
}

func (h *ChatHandler) HandleDeleteSession(c *gin.Context) {
	sessionID := c.Param("id")

	if err := h.sessionManager.DeleteSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session deleted"})
}

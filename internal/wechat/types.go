package wechat

type WebhookMessage struct {
	WebhookURL     string      `json:"WebhookUrl"`
	ChatID         string      `json:"ChatId"`
	ChatType       string      `json:"ChatType"`
	GetChatInfoURL string      `json:"GetChatInfoUrl"`
	From           MessageFrom `json:"From"`
	MsgID          string      `json:"MsgId"`
	MsgType        string      `json:"MsgType"`
	Text           MessageText `json:"Text"`
	Event          EventInfo   `json:"Event,omitempty"`
}

type MessageFrom struct {
	UserID string `json:"UserId"`
	Name   string `json:"Name"`
	Alias  string `json:"Alias"`
}

type MessageText struct {
	Content string `json:"Content"`
}

type EventInfo struct {
	EventType    string `json:"EventType"`
	ActionUserID string `json:"ActionUserId,omitempty"`
}

type WebhookResponse struct {
	MsgType string             `json:"msgtype"`
	Text    WebhookTextContent `json:"text"`
}

type WebhookTextContent struct {
	Content string `json:"content"`
}

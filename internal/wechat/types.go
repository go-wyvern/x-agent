package wechat

type EncryptedRequest struct {
	Encrypt string `json:"encrypt"`
}

type IncomingMessage struct {
	MsgID      string      `json:"msgid"`
	CreateTime int64       `json:"create_time,omitempty"`
	AIBotID    string      `json:"aibotid"`
	ChatID     string      `json:"chatid,omitempty"`
	ChatType   string      `json:"chattype,omitempty"`
	From       *FromInfo   `json:"from,omitempty"`
	MsgType    string      `json:"msgtype"`
	Text       *TextMsg    `json:"text,omitempty"`
	Image      *ImageMsg   `json:"image,omitempty"`
	Stream     *StreamMsg  `json:"stream,omitempty"`
	Event      *EventMsg   `json:"event,omitempty"`
	Mixed      *MixedMsg   `json:"mixed,omitempty"`
}

type TextMsg struct {
	Content string `json:"content"`
}

type ImageMsg struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type StreamMsg struct {
	ID     string `json:"id"`
	Finish bool   `json:"finish,omitempty"`
}

type FromInfo struct {
	CorpID string `json:"corpid,omitempty"`
	UserID string `json:"userid"`
}

type EventMsg struct {
	EventType         string                 `json:"eventtype"`
	TemplateCardEvent *TemplateCardEventData `json:"template_card_event,omitempty"`
}

type TemplateCardEventData struct {
	CardType      string         `json:"card_type"`
	EventKey      string         `json:"event_key"`
	TaskID        string         `json:"task_id"`
	SelectedItems *SelectedItems `json:"selected_items,omitempty"`
}

type SelectedItems struct {
	SelectedItem []SelectedItem `json:"selected_item"`
}

type SelectedItem struct {
	QuestionKey string    `json:"question_key"`
	OptionIDs   *OptionID `json:"option_ids"`
}

type OptionID struct {
	OptionID []string `json:"option_id"`
}

type MixedMsg struct {
	MsgItem []MixedMsgItem `json:"msg_item"`
}

type MixedMsgItem struct {
	MsgType string    `json:"msgtype"`
	Text    *TextMsg  `json:"text,omitempty"`
	Image   *ImageMsg `json:"image,omitempty"`
}

type TextResponse struct {
	MsgType string   `json:"msgtype"`
	Text    *TextMsg `json:"text"`
}

type StreamResponse struct {
	MsgType string             `json:"msgtype"`
	Stream  StreamResponseData `json:"stream"`
}

type StreamResponseData struct {
	ID      string          `json:"id"`
	Finish  bool            `json:"finish"`
	Content string          `json:"content,omitempty"`
	MsgItem []StreamMsgItem `json:"msg_item,omitempty"`
}

type StreamMsgItem struct {
	MsgType string           `json:"msgtype"`
	Text    *TextMsg         `json:"text,omitempty"`
	Image   *ImageMsgItem    `json:"image,omitempty"`
}

type ImageMsgItem struct {
	Base64 string `json:"base64"`
	MD5    string `json:"md5"`
}

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

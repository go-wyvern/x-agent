package wechat

type Config struct {
	CorpID         string `json:"corp_id"`
	AgentID        int    `json:"agent_id"`
	Secret         string `json:"secret"`
	Token          string `json:"token"`
	EncodingAESKey string `json:"encoding_aes_key"`
}

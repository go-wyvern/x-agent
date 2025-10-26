package wechat

import "encoding/xml"

type Message struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        int64    `xml:"MsgId"`
	AgentID      int      `xml:"AgentID"`
}

type TextResponse struct {
	ToUser  string      `json:"touser"`
	MsgType string      `json:"msgtype"`
	AgentID int         `json:"agentid"`
	Text    TextContent `json:"text"`
}

type TextContent struct {
	Content string `json:"content"`
}

type APIResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

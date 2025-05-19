package signal

import "encoding/json"

type Request struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth,omitempty"`
}

type Response struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
	Data    string `json:"data,omitempty"`
}

type Message struct {
	Type     string          `json:"type"`
	ClientID string          `json:"client_id"`
	Payload  json.RawMessage `json:"payload"`
	// Auth string `json:"auth"`
}
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

type Payload interface {
	GetClientID() string
}

type OfferPayload struct {
	SDP string `json:"sdp"`
}

type AnswerPayload struct {
	SDP string `json:"sdp"`
}

type IceCandidatePayload struct {
	Candidate string `json:"candidate"`
	SDPMid string `json:"sdpMid"`
	SDPMLineIndex uint16 `json:"sdpMLineIndex"`
}

type SFUMessage interface {
	GetClientID() string
}

type SFUOffer struct{
	ClientID string 
	Payload OfferPayload
}
func (o *SFUOffer) GetClientID() string { return o.ClientID } 

type SFUAnswer struct{
	ClientID string
	Payload AnswerPayload
}
func (a *SFUAnswer) GetClientID() string { return a.ClientID }

type SFUIceCandidate struct{
	ClientID string
	Payload IceCandidatePayload
}
func (i *SFUIceCandidate) GetClientID() string { return i.ClientID }

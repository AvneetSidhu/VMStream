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
	ClientID string          `json:"clientId"`
	Payload  json.RawMessage `json:"payload"`
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
	SDPMid string `json:"sdpMid,omitempty"`
	SDPMLineIndex uint16 `json:"sdpMLineIndex,omitempty"`
}

type SFUMessage interface {
	GetClientID() string
	GetType() string
}

type SFUOffer struct{
	ClientID string `json:"clientId"`
	Type string `json:"type"`
	Payload OfferPayload `json:"payload"`
}
func (o *SFUOffer) GetClientID() string { return o.ClientID } 
func (o *SFUOffer) GetType() string { return o.Type }
type SFUAnswer struct{
	ClientID string `json:"clientId"`
	Type string `json:"type"`
	Payload AnswerPayload `json:"payload"`
}
func (a *SFUAnswer) GetClientID() string { return a.ClientID }
func (a *SFUAnswer) GetType() string { return a.Type }

type SFUIceCandidate struct{
	ClientID string `json:"clientId"`
	Type string `json:"type"`
	Payload IceCandidatePayload `json:"payload"`
}
func (i *SFUIceCandidate) GetClientID() string { return i.ClientID }
func (i *SFUIceCandidate) GetType() string { return i.Type }

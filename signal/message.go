package main

import (
	"fmt"
	"bytes"
	"encoding/json"
)

type Message struct {
	Type string `json:"type"`
	ClientID string `json:"client_id"`
	Payload string `json:"payload"`
}

func decodeMessage(msgBytes []byte) (Message, error) {
	var msg Message
	decoder := json.NewDecoder(bytes.NewReader(msgBytes))	
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&msg)
	return msg, err
}	

func checkFields(msg Message) (error) {
	if msg.Type == "" {
		return fmt.Errorf("missing or empty field: type")
	}
	if msg.ClientID == "" {
		return fmt.Errorf("missing or empty field: client_id")
	}
	if msg.Payload == "" {
		return fmt.Errorf("missing or empty field: payload")
	}
	return nil
}
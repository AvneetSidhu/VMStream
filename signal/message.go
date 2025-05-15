package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Message struct {
	Type string `json:"type"`
	ClientID string `json:"client_id"`
	Payload json.RawMessage `json:"payload"`
	Auth string `json:"auth"`
}

func createErrorMessage(err error) (Message) {
	return Message{
		Type: "error",
		ClientID: "",
		Payload: json.RawMessage(fmt.Sprintf(`{"error": "%s"}`, err.Error())),
		Auth: "",
	}
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
	if len(msg.Payload) == 0 {
		return fmt.Errorf("missing or empty field: payload")
	}
	if msg.Auth == "" {
		return fmt.Errorf("missing or empty field: auth")
	}
	return nil
}

func validateClientToken(clientID string, clientToken string) (bool) {
	extractedUsername, err := validateJWTToken(clientToken)
	if err != nil {
		fmt.Println("Error validating client token:", err)
		return false
	}

	if clientID != extractedUsername {
		fmt.Println("Client ID does not match token username for user:", clientID)
		return false
	}
	return true && userExists(clientID)
}
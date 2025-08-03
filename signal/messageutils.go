package signal

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func createErrorMessage(err error) (Message) {
	return Message{
		Type: "error",
		ClientID: "",
		Payload: json.RawMessage(fmt.Sprintf(`{"error": "%s"}`, err.Error())),
	}
}


func decodeMessage(msgBytes []byte) (Message, error) {
	var msg Message
	decoder := json.NewDecoder(bytes.NewReader(msgBytes))	
	decoder.DisallowUnknownFields() 
	err := decoder.Decode(&msg)
	return msg, err
}	

func decodePayload[T any](payloadBytes []byte, out *T) error {
	return json.Unmarshal(payloadBytes, out)
}


func checkFields(msg Message) (error) {
	if msg.Type == "" {
		return fmt.Errorf("missing or empty field: type")
	}

	if msg.ClientID == "" {
		return fmt.Errorf("missing or empty field: clientId")
	}

	if len(msg.Payload) == 0 {
		return fmt.Errorf("missing or empty field: payload")
	}

	return nil
}


package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"encoding/json"
	"bytes"
)

type Message struct {
	Type string `json:"type"`
	ClientID string `json:"client_id"`
	Payload string `json:"payload"`
}

var upgrader = websocket.Upgrader {
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()    

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		// fmt.Printf("Received message: %s\n", msgBytes)

		var raw map[string]interface{}

		if err := json.Unmarshal(msgBytes, &raw); err != nil {
			fmt.Println("invalid JSON:", err)
			break
		}

		for _, field := range []string{"type", "client_id", "payload"} {
			if _, ok := raw[field]; !ok {
				fmt.Printf("missing field: %s\n", field)
				return
			}
		}

		var msg Message

		decoder := json.NewDecoder(bytes.NewReader(msgBytes))	
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&msg); err != nil {
			fmt.Println("Struct parse error:", err)
			break
		}

		// fmt.Printf("Received message: Type=%s, ClientID=%s, Payload=%s\n", msg.Type, msg.ClientID, msg.Payload)

		err = conn.WriteMessage(websocket.TextMessage, msgBytes)
		if err != nil {
			fmt.Println("Error writing message:", err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	fmt.Println("Server started at :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
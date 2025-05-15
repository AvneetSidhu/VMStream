package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	mu      sync.RWMutex
	clients = make(map[string]*websocket.Conn) // map of client_id to connection
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func addClient(clientID string, conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()
	clients[clientID] = conn
}

func removeClient(clientID string) {
	mu.Lock()
	defer mu.Unlock()
	if conn, ok := clients[clientID]; ok {
		conn.Close()
		delete(clients, clientID)
	}
}

func handleMessage(msg Message) error {
	// msgType, clientId, payload := msg.Type, msg.ClientID, msg.Payload
	return nil
	// switch type {
	// case "offer":
	// case "answer":
	// case "ice_candidate":
	// }
}

func messageLoopCleanup(conn *websocket.Conn) {
	defer conn.Close()
	clientID := conn.RemoteAddr().String()
	removeClient(clientID) // remove client from connection manager
	fmt.Println("Client disconnected:", clientID)
}

func pionReadLoop(conn *websocket.Conn) {
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		decodedMsg, decodeErr := decodeMessage(msgBytes) // decode message
		if decodeErr != nil {
			fmt.Println("Error decoding message:", decodeErr)
			break
		}

		fieldsErr := checkFields(decodedMsg) // check message is correctly formed before decoding
		if fieldsErr != nil {
			fmt.Println("Error checking fields:", fieldsErr)
			break
		}

		fmt.Printf("Received message: %+v\n\n", decodedMsg)

		err = handleMessage(decodedMsg) // handle message
		if err != nil {
			fmt.Println("Error handling message:", err)
			break
		}
	}
}

func pionWriteLoop(conn *websocket.Conn) {
	for {
		// Here you would typically send messages to the client
		// For example, sending a ping message every 10 seconds
		err := conn.WriteMessage(websocket.PingMessage, []byte("ping"))
		if err != nil {
			fmt.Println("Error writing message:", err)
			break
		}
		time.Sleep(10 * time.Second)
	}
}

func pionMessageLoop(conn *websocket.Conn) {
	go pionReadLoop(conn) // start Pion read loop
	go pionWriteLoop(conn) // start Pion write loop
	defer messageLoopCleanup(conn) // cleanup on exit
}

func clientMessageLoop(conn *websocket.Conn) {
	defer messageLoopCleanup(conn) // cleanup on exit
	for {
		_, msgBytes, err := conn.ReadMessage()

		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		decodedMsg, decodeErr := decodeMessage(msgBytes) // decode message

		if decodeErr != nil {
			fmt.Println("Error decoding message:", decodeErr)
			errorMsgBytes, _ := json.Marshal(createErrorMessage(decodeErr))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		fieldsErr := checkFields(decodedMsg) // check message is correctly formed before decoding

		if fieldsErr != nil {
			fmt.Println("Error checking fields:", fieldsErr)
			errorMsgBytes, _ := json.Marshal(createErrorMessage(fieldsErr))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		fmt.Printf("Received message: %+v\n\n", decodedMsg)

		handlingErr := handleMessage(decodedMsg) // handle message

		if handlingErr != nil {
			fmt.Println("Error handling message:", handlingErr)
			errorMsgBytes, _ := json.Marshal(createErrorMessage(handlingErr))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}
	}
}

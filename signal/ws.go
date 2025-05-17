package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

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

func broadcastMessage() {
	for msg := range fromPion {
		mu.RLock()
		conn, ok := clients[msg.ClientID]
		mu.RUnlock()
		if !ok {
			fmt.Println("Client not found:", msg.ClientID)
			continue
		}

		if err := conn.WriteJSON(msg); err != nil {
			fmt.Printf("Error writing message to client %s: %v\n", msg.ClientID, err)
		}
	}
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

		if !validateClientToken(decodedMsg.ClientID, decodedMsg.Auth) {
			fmt.Println("Invalid client token for user:", decodedMsg.ClientID)
			errorMsgBytes, _ := json.Marshal(createErrorMessage(fmt.Errorf("invalid client token")))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		fmt.Printf("Received message: %+v\n\n", decodedMsg)

		handleErr := handleMessage(decodedMsg) // handle message
		if handleErr != nil {
			fmt.Println("Error handling message:", err)
			break
		}
		fromPion <- &decodedMsg // broadcast to clients 
	}
}

func pionWriteLoop(conn *websocket.Conn) {
	for msg := range toPion {
		if err := conn.WriteJSON(msg); err != nil {
			fmt.Println("Error writing message:", err)
			break
		}
	}
}

func pionMessageLoop(conn *websocket.Conn) {
	go pionReadLoop(conn) // start Pion read loop
	go pionWriteLoop(conn) // start Pion write loop
	go broadcastMessage() // start broadcast loop 
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

		// fmt.Printf("Received message: %+v\n\n", decodedMsg)
		if !validateClientToken(decodedMsg.ClientID, decodedMsg.Auth) {
			fmt.Println("Invalid client token for user:", decodedMsg.ClientID)
			errorMsgBytes, _ := json.Marshal(createErrorMessage(fmt.Errorf("invalid client token")))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		handlingErr := handleMessage(decodedMsg) // handle message

		if handlingErr != nil {
			fmt.Println("Error handling message:", handlingErr)
			errorMsgBytes, _ := json.Marshal(createErrorMessage(handlingErr))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		toPion <- &decodedMsg // broadcast to Pion
	}
}

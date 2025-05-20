package signal

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
	fmt.Println("Removing client:", clientID)
	mu.Lock()
	defer mu.Unlock()
	if conn, ok := clients[clientID]; ok {
		conn.Close()
		delete(clients, clientID)
	}
}

func handleMessage(msg Message)(SFUMessage, error) {
	switch msg.Type {
	case "offer":
		var payload OfferPayload
		if err := decodePayload(msg.Payload, &payload); err != nil {
			return nil, err
		}
		fmt.Printf("Received offer from client %s: %s\n", msg.ClientID, payload.SDP)
		return &SFUOffer{ClientID: msg.ClientID, Payload: payload}, nil
	case "answer":
		var payload AnswerPayload
		if err := decodePayload(msg.Payload, &payload); err != nil {
			return nil, err
		}
		fmt.Printf("Received answer from client %s: %s\n", msg.ClientID, payload.SDP)
		return &SFUAnswer{ClientID: msg.ClientID, Payload: payload}, nil
	case "ice_candidate":
		var payload IceCandidatePayload
		if err := decodePayload(msg.Payload, &payload); err != nil {
			return nil, err
		}
		fmt.Printf("Received ICE candidate from client %s: %s\n", msg.ClientID, payload.Candidate)
		return &SFUIceCandidate{ClientID: msg.ClientID, Payload: payload}, nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func messageLoopCleanup(conn *websocket.Conn) {
	defer conn.Close()
	clientID := conn.RemoteAddr().String()
	removeClient(clientID) // remove client from connection manager
	fmt.Println("Client disconnected:", clientID)
}

func sendMessage(conn *websocket.Conn, msg Message) {
	err := conn.WriteJSON(msg)
	if err != nil {
		fmt.Println("Error broadcasting message:", err)
	}
}

func pionReadLoop() {
	for msg := range FromSFU {
		mu.RLock()
		conn, ok := clients[msg.ClientID]
		mu.RUnlock()
		if !ok {
			fmt.Printf("Client %s not connected\n", msg.ClientID)
			continue
		}
		sendMessage(conn, *msg) // broadcast message to all clients
	}
}

func StartSFUMessageLoop() {
	go pionReadLoop() // start Pion read loop
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

		toForward, handlingErr := handleMessage(decodedMsg) // handle message

		if handlingErr != nil {
			fmt.Println("Error handling message:", handlingErr)
			errorMsgBytes, _ := json.Marshal(createErrorMessage(handlingErr))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		} // broadcast to Pion

		ToSFU <- toForward
	}
}

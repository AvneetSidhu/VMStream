package signal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	mu      sync.RWMutex
	clients = make(map[string]*websocket.Conn) 
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

func handleMessage(msg Message)(SFUMessage, error) {
	switch msg.Type {
	case "offer":
		var payload OfferPayload
		if err := decodePayload(msg.Payload, &payload); err != nil {
			return nil, err
		}
		logger.Info("Received offer from client", zap.String("client_id", msg.ClientID), zap.String("sdp", payload.SDP))
		return &SFUOffer{ClientID: msg.ClientID, Type:"offer", Payload: payload}, nil
	case "answer":
		var payload AnswerPayload
		if err := decodePayload(msg.Payload, &payload); err != nil {
			return nil, err
		}
		logger.Info("Received answer from client", zap.String("client_id", msg.ClientID), zap.String("sdp", payload.SDP))
		return &SFUAnswer{ClientID: msg.ClientID, Type: "answer", Payload: payload}, nil
	case "ice-candidate":
		var payload IceCandidatePayload
		if err := decodePayload(msg.Payload, &payload); err != nil {
			return nil, err
		}
		logger.Info("Received ICE candidate from client", zap.String("client_id", msg.ClientID), zap.String("candidate", payload.Candidate))
		return &SFUIceCandidate{ClientID: msg.ClientID, Type: "ice-candidate", Payload: payload}, nil
	case "termination":
		logger.Info("Client disconnected from signaling websocket, WebRTC Connection Success!", zap.String("client_id", msg.ClientID))
		return nil, nil
	default:
		logger.Warn("Unknown message type", zap.String("type", msg.Type))
		return nil, fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func messageLoopCleanup(clientID string) {
	removeClient(clientID)
	logger.Info("Client disconnected from signaling websocket", zap.String("client_id", clientID))
}

func sendMessage(conn *websocket.Conn, msg SFUMessage) {
	switch msg := msg.(type) {
	case *SFUIceCandidate:
		conn.WriteJSON(msg)
	case *SFUAnswer:
		conn.WriteJSON(msg)
	default:
		logger.Warn("Unknown Message Type")
	}
}

func sfuReadLoop() {
	for msg := range FromSFU {
		mu.RLock()
		conn, ok := clients[msg.GetClientID()]
		mu.RUnlock()
		if !ok {
			logger.Warn("Client not connected", zap.String("client_id", msg.GetClientID()))
			continue
		}

		sendMessage(conn, msg)
	}
}

func StartSFUMessageLoop() {
	go sfuReadLoop() 
}

func clientMessageLoop(conn *websocket.Conn, clientID string) {
	defer messageLoopCleanup(clientID)
	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			logger.Error("Error reading message from client", zap.String("client_id", clientID), zap.Error(err))
			break
		}

		decodedMsg, decodeErr := decodeMessage(msgBytes)

		if decodeErr != nil {
			logger.Error("Error decoding message", zap.Error(decodeErr))
			errorMsgBytes, _ := json.Marshal(createErrorMessage(decodeErr))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		fieldsErr := checkFields(decodedMsg) 
		if fieldsErr != nil {
			logger.Error("Error checking fields in message", zap.Error(fieldsErr))
			errorMsgBytes, _ := json.Marshal(createErrorMessage(fieldsErr))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		toForward, handlingErr := handleMessage(decodedMsg) 
		if handlingErr != nil {
			logger.Error("Error handling message", zap.Error(handlingErr))
			errorMsgBytes, _ := json.Marshal(createErrorMessage(handlingErr))
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}
		
		if toForward == nil {
			break
		}

		ToSFU <- toForward
	}
}

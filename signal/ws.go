package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleMessage(msg Message) error {
	// msgType, clientId, payload := msg.Type, msg.ClientID, msg.Payload
	return nil
	// switch type {
	// case "register":
	// 	registerMsg(msg)
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

func messageLoop(conn *websocket.Conn) {
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
			errorMsg := Message{
				Type:     "error",
				ClientID: "",
				Payload:  "Error decoding message:" + decodeErr.Error(),
			}
			errorMsgBytes, _ := json.Marshal(errorMsg)
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		fieldsErr := checkFields(decodedMsg) // check message is correctly formed before decoding

		if fieldsErr != nil {
			fmt.Println("Error checking fields:", fieldsErr)
			errorMsg := Message{
				Type:     "error",
				ClientID: "",
				Payload:  "Error checking fields: " + fieldsErr.Error(),
			}
			errorMsgBytes, _ := json.Marshal(errorMsg)
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		fmt.Printf("Received message: %+v\n\n", decodedMsg)

		handlingErr := handleMessage(decodedMsg) // handle message

		if handlingErr != nil {
			fmt.Println("Error handling message:", handlingErr)
			errorMsg := Message{
				Type:     "error",
				ClientID: decodedMsg.ClientID,
				Payload:  "Error handling message: " + handlingErr.Error(),
			}
			errorMsgBytes, _ := json.Marshal(errorMsg)
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}
	}
}

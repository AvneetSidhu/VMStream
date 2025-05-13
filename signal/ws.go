package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"encoding/json"
)

var upgrader = websocket.Upgrader {
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func connect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading connection:", err)
		return
	}
	addClient(r.URL.Query().Get("client_id"), conn) // add client to connection manager
	fmt.Println("Client connected:", r.URL.Query().Get("client_id"))
	messageLoop(conn) // start message loop
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
				Type: "error",
				ClientID: "",
				Payload: "Error decoding message:" + decodeErr.Error(),
			}
			errorMsgBytes, _ := json.Marshal(errorMsg)
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}

		fieldsErr := checkFields(decodedMsg) //check message is correctly formed before decoding

		if fieldsErr != nil {
			fmt.Println("Error checking fields:", fieldsErr)
			errorMsg := Message{
				Type: "error",
				ClientID: "",
				Payload: "Error checking fields: " + fieldsErr.Error(),
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
				Type: "error",
				ClientID: decodedMsg.ClientID,
				Payload: "Error handling message: " + handlingErr.Error(),
			}
			errorMsgBytes, _ := json.Marshal(errorMsg)
			conn.WriteMessage(websocket.TextMessage, errorMsgBytes)
			break
		}
	}
}


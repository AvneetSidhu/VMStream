package main

import (
	"fmt"
	"net/http"
	"os"
)

var jwtsecret = os.Getenv("JWT_SECRET")
var log_mode = os.Getenv("LOG_MODE")

func handleMessage (msg Message) error {
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

func main() {
	// http.HandleFunc("/register", )
	http.HandleFunc("/connect", connect)
	fmt.Println("Server started at :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
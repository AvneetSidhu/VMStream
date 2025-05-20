package main

import (
	"fmt"
	"net/http"
	"os"
	sfu "webrtc-gateway/SFU"
	"webrtc-gateway/signal"

	"github.com/joho/godotenv"
)

var PION_NAME string
var JWT_SECRET string


func main() {
	godotenv.Load()
	fmt.Println("Loading environment variables...")
	JWT_SECRET = os.Getenv("JWT_SECRET")
	PION_NAME = os.Getenv("PION_NAME")

	signal.SetJWTSecret(JWT_SECRET)
	signal.SetPionName(PION_NAME)
	go signal.StartSFUMessageLoop()

	sfu.FromSignal = signal.ToSFU
	go sfu.Start()

	http.HandleFunc("/login", signal.LoginHandler)
	http.HandleFunc("/connect", signal.ClientConnectHandler)
	// http.HandleFunc("/login", signal.LoginHandler)
	http.HandleFunc("/register", signal.RegisterHandler)
	fmt.Println("Server started at :8080")
	
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

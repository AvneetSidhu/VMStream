package main

import (
	"sync"
	"github.com/gorilla/websocket"
)

var mu sync.Mutex
var clients = make(map[string] *websocket.Conn) // map of client_id to connection

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
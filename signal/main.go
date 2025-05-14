package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Body struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

type Response struct {
	Message string `json:"message"`
}

func writeJSONError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Message: err.Error()})
}

func decodeBody(r *http.Request) (Body, error) {
	var body Body
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return Body{}, err
	}
	return body, nil
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

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}

	body, err := decodeBody(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("bad request"))
		return
	}

	username, password := body.Username, body.Password
	fmt.Println("Log in request from:", username)

	userExists := userExists(username)
	if !userExists {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return
	}

	hashedPassword, err := getHashedPassword(username)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	if !checkPasswordHash(password, hashedPassword) {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return
	}

	fmt.Println("Login successful, generating JWT for:", username)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Message: "Login successful"})
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}

	body, err := decodeBody(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("bad request"))
		return
	}

	username, password := body.Username, body.Password

	if userExists(username) {
		writeJSONError(w, http.StatusConflict, fmt.Errorf("user already exists"))
		return
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	err = storeUser(username, hashedPassword)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	fmt.Println("User registered successfully:", username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "Registration successful"})
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/connect", connect)
	http.HandleFunc("/login", login)
	http.HandleFunc("/register", register)
	fmt.Println("Server started at :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

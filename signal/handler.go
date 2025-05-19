package signal

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func writeJSONError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Message: err.Error()})
}

func decodeBody(r *http.Request) (Request, error) {
	var body Request
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return Request{}, err
	}
	if body.Username == "" || body.Password == "" {
		return Request{}, fmt.Errorf("missing required fields: username or password")
	}
	return body, nil
}

func PionConnectHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	auth := r.URL.Query().Get("auth")
	if !validatePionToken(auth) {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("invalid auth token"))
		return
	}

	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading connection:", err)
		return
	}

	pionMessageLoop(conn) // start Pion message loop
	fmt.Println("Pion connected:", clientID)
}

func ClientConnectHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	auth := r.URL.Query().Get("auth")
	if clientID == "" || auth == "" {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("missing required fields: client_id or auth"))
		return
	}

	if !validateClientToken(clientID, auth) {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("invalid auth token"))
		return
	}

	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading connection:", err)
		return
	}
	addClient(r.URL.Query().Get("client_id"), conn) // add client to connection manager
	fmt.Println("Client connected:", r.URL.Query().Get("client_id"))
	go clientMessageLoop(conn) // start message loop
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
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
	token, err := generateJWTToken(username)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
	}
	json.NewEncoder(w).Encode(Response{Message: "Login successful", Token: token})
	fmt.Println("JWT token generated for user:", username)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
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
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Message: "Registration successful"})
}

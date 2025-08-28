package signal

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
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

func ClientConnectHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	auth := r.URL.Query().Get("auth")
	if clientID == "" || auth == "" {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("missing required fields: client_id or auth"))
		logger.Error("Missing required fields in query parameters", zap.String("client_id", clientID), zap.String("auth", auth))
		return
	}

	if !validateClientToken(clientID, auth) {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("invalid auth token"))
		logger.Error("Invalid auth token for client", zap.String("client_id", clientID))
		return
	}

	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		logger.Warn("Method not allowed for client connection", zap.String("method", r.Method))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		logger.Error("Error upgrading connection", zap.Error(err))
		return
	}

	addClient(clientID, conn)

	logger.Info("Signaling started for client", zap.String("client_id", clientID))
	go clientMessageLoop(conn, clientID)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		logger.Warn("Method not allowed for login", zap.String("method", r.Method))
		return
	}

	body, err := decodeBody(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("bad request"))
		logger.Error("Error decoding login request body", zap.Error(err))
		return
	}

	username, password := body.Username, body.Password

	userFound := userExists(username)
	if !userFound {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		logger.Warn("User does not exist", zap.String("username", username))
		return
	}

	hashedPassword, err := getHashedPassword(username)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		logger.Error("Error retrieving hashed password", zap.String("username", username), zap.Error(err))
		return
	}

	if !checkPasswordHash(password, hashedPassword) {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		logger.Warn("Invalid password for user", zap.String("username", username))
		return
	}
	
	token, err := generateJWTToken(username)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		logger.Error("Error generating JWT token", zap.String("username", username), zap.Error(err))
		return
	}

	logger.Info("User logged in successfully", zap.String("username", username))

	cookie, err := issueRefreshTokenCookie(username)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		logger.Error("Error issuing refresh token cookie", zap.String("username", username), zap.Error(err))
		return
	}

	http.SetCookie(w, &cookie)
	json.NewEncoder(w).Encode(Response{Message: "Login successful", Token: token})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		logger.Warn("Method not allowed for registration", zap.String("method", r.Method))
		return
	}

	body, err := decodeBody(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Errorf("bad request"))
		logger.Warn("Error decoding registration request body", zap.Error(err))
		return
	}

	username, password := body.Username, body.Password

	if userExists(username) {
		writeJSONError(w, http.StatusConflict, fmt.Errorf("user already exists"))
		logger.Warn("User already exists", zap.String("username", username))
		return
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		logger.Error("Error hashing password", zap.String("username", username), zap.Error(err))
		return
	}

	err = storeUser(username, hashedPassword)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		logger.Error("Error storing user", zap.String("username", username), zap.Error(err))
		return
	}

	logger.Info("User registered successfully", zap.String("username", username))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Message: "Registration successful"})
}


func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
		logger.Warn("Method not allowed for token refresh", zap.String("method", r.Method))
		return
	}

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		logger.Warn("Refresh token cookie not found", zap.Error(err))
		return
	}

	refreshToken := cookie.Value

	extractedUsername, err := validateJWTToken(refreshToken)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, fmt.Errorf("invalid refresh token"))
		logger.Warn("Error validating refresh token", zap.Error(err))
		return
	}

	token, err := generateJWTToken(extractedUsername)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		logger.Error("Error generating new JWT token", zap.String("username", extractedUsername), zap.Error(err))
		return
	}
	logger.Info("Refresh token validated, new token issued", zap.String("username", extractedUsername))

	json.NewEncoder(w).Encode(Response{Message: "Token refreshed successfully", Token: token, Data: extractedUsername})
}
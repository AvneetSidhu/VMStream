package signal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func getHashedPassword(username string) (string, error) {
	user, err := getUser(username)
	if err != nil {
		logger.Error("Error getting user:", zap.Error(err))
		return "", err
	}
	return user, nil
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateJWTToken(clientID string) (string, error) {
	claims := jwt.MapClaims{
		"client_id": clientID,
		"exp":      jwt.NewNumericDate(time.Now().Add(time.Minute * 15)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtSecret))
}

func issueRefreshTokenCookie(clientID string) (http.Cookie, error) {
	refreshTokenClaims := jwt.MapClaims{
		"client_id": clientID,
		"exp":       jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(jwtSecret))

	if err != nil {
		return http.Cookie{}, err
	}

	cookie := http.Cookie{
		Name:     "refresh_token",
		Value:   refreshTokenString,
		Path:    "/api/refresh",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	}

	return cookie, nil
}

func validateJWTToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			err := fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			logger.Error("JWT signing method error", zap.Error(err))
			return nil, err
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		logger.Error("Error parsing JWT token", zap.Error(err))
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		clientID := claims["client_id"].(string)
		return clientID, nil
	}

	err = fmt.Errorf("invalid token")
	logger.Error("JWT token invalid", zap.Error(err))
	return "", err
}

func validateClientToken(clientID string, clientToken string) bool {
	extractedUsername, err := validateJWTToken(clientToken)
	if err != nil {
		logger.Error("Error validating client token", zap.Error(err))
		return false
	}

	if clientID != extractedUsername {
		logger.Warn("Client ID does not match token username", zap.String("clientID", clientID), zap.String("extractedUsername", extractedUsername))
		return false
	}

	return userExists(clientID)
}

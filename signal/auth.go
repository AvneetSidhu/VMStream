package signal

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
		fmt.Println("Error getting user:", err)
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
		"exp":      jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtSecret))
}

func validateJWTToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		clientID := claims["client_id"].(string)
		return clientID, nil
	}

	return "", fmt.Errorf("invalid token")
}


func validateClientToken(clientID string, clientToken string) (bool) {
	extractedUsername, err := validateJWTToken(clientToken)
	if err != nil {
		fmt.Println("Error validating client token:", err)
		return false
	}

	if clientID != extractedUsername {
		fmt.Println("Client ID does not match token username for user:", clientID)
		return false
	}
	return true && userExists(clientID)
}

func validatePionToken(clientToken string) (bool) {
	extractedUsername, err := validateJWTToken(clientToken)
	if err != nil {
		fmt.Println("Error validating client token:", err)
		return false
	}

	if pionName != extractedUsername {
		fmt.Println("Pion connection auth token does not match Pion name for user:", clientToken)
		return false
	}
	return true
}
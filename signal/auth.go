package main

import (
	"fmt"
	"os"
	"golang.org/x/crypto/bcrypt"
)

var jwtsecret = os.Getenv("JWT_SECRET")

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
	if err != nil {
		return false
	}
	return true
}
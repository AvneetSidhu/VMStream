package signal

import (
	"encoding/csv"
	"fmt"
	"os"

	"go.uber.org/zap"
)

func storeUser(username, hashedPassword string) error {
	file, err := os.OpenFile("db.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()

	if err != nil {
		logger.Error("Error opening file:", zap.Error(err))
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{username, hashedPassword})

	return err
}

func userExists(username string) bool {
	file, err := os.Open("db.csv")
	if err != nil {
		logger.Error("Error opening file:", zap.Error(err))
		return false
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		logger.Error("Error reading file:", zap.Error(err))
		return false
	}

	for _, record := range records {
		if record[0] == username {
			return true
		}
	}

	return false
}

func getUser(username string) (string, error) {
	file, err := os.Open("db.csv")
	if err != nil {
		logger.Error("Error opening file:", zap.Error(err))
		return "", err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		logger.Error("Error reading file:", zap.Error(err))
		return "", err
	}

	for _, record := range records {
		if record[0] == username {
			return record[1], nil
		}
	}

	return "", fmt.Errorf("user not found")
}
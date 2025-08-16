package signal

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3" // Import SQLite driver
	"go.uber.org/zap"
)

var db *sql.DB

func InitDB() {
	var err error
	db, err = sql.Open("sqlite3", "./test.db")

	if err != nil {
		logger.Fatal("Failed to open database:", zap.Error(err))
	}

	tx, err := db.Begin()

	if err != nil {
		logger.Fatal("Failed to begin transaction:", zap.Error(err))
	}

	createUsersTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		hashed_password TEXT NOT NULL
	);`

	createLoginHistoryTableQuery := `
	CREATE TABLE IF NOT EXISTS login_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    login_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	if _, err = tx.Exec(createUsersTableQuery); err != nil {
		tx.Rollback()
		logger.Fatal("Failed to create users table:", zap.Error(err))
	}

	if _, err = tx.Exec(createLoginHistoryTableQuery); err != nil {
		tx.Rollback()
		logger.Fatal("Failed to create login history table:", zap.Error(err))
	}

	if err = tx.Commit(); err != nil {
		logger.Fatal("Failed to commit transaction:", zap.Error(err))
	}
}

func storeUser(username, hashedPassword string) error {
	_, err := db.Exec(`INSERT INTO users (username, hashed_password) VALUES (?, ?)`, username, hashedPassword)

	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.username" {
			return sql.ErrNoRows // User already exists
		}
		return err
	}

	return nil
}

func userExists(username string) bool {
	var id int

	err := db.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			return false 
		}
	}

	return true
}

func getUser(username string) (string, error) {
	var hashedPassword string

	err := db.QueryRow(`SELECT hashed_password FROM users WHERE username = ?`, username).Scan(&hashedPassword)

	if err != nil {
		return "", err
	}

	return hashedPassword, nil
}
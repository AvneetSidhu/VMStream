package main

import (
	"net/http"
	"os"
	"strings"
	sfu "webrtc-gateway/sfu"
	"webrtc-gateway/signal"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)


func startSignalServer(jwtSecret string, logger *zap.Logger) {
	signal.SetJWTSecret(jwtSecret)
	signal.SetLogger(logger)

	go signal.StartSFUMessageLoop()

	sfu.FromSignal = signal.ToSFU
	sfu.ToSignal = signal.FromSFU

	http.HandleFunc("/login", signal.LoginHandler)
	http.HandleFunc("/connect", signal.ClientConnectHandler)
	http.HandleFunc("/register", signal.RegisterHandler)
}

func startSFU(logger *zap.Logger) {
	sfu.SetLogger(logger)
	go sfu.Start()
}

func InitLogger() *zap.Logger {
	levelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))
	var level zapcore.Level

	switch levelStr {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn", "warning":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel // default if unspecified or invalid
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(level)

	var err error
	logger, err := cfg.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	return logger
}

func main() {
	logger := InitLogger()
	defer logger.Sync()

	if err := godotenv.Load(); err != nil {
		logger.Error("Error loading .env file", zap.Error(err))
		return
	}

	logger.Info("Starting SFU Server...")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET is not set in .env file")
		return
	}

	startSignalServer(jwtSecret, logger)
	startSFU(logger)
	
	logger.Info("SFU Server is running on port 8080")
	
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}


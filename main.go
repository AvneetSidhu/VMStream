package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	sfu "webrtc-gateway/sfu"
	"webrtc-gateway/signal"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func startClient() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
}

func startSignalServer(jwtSecret string, logger *zap.Logger) {
	signal.SetJWTSecret(jwtSecret)
	signal.SetLogger(logger)
	signal.InitDB()
	go signal.StartSFUMessageLoop()

	sfu.FromSignal = signal.ToSFU
	sfu.ToSignal = signal.FromSFU

	http.HandleFunc("/api/login", signal.LoginHandler)
	http.HandleFunc("/api/connect", signal.ClientConnectHandler)
	http.HandleFunc("/api/register", signal.RegisterHandler)
	http.HandleFunc("/api/refresh", signal.RefreshHandler)
}

func startSFU(logger *zap.Logger, tailnet string) {
	sfu.SetLogger(logger)
	widthStr := os.Getenv("WIDTH")
	heightStr := os.Getenv("HEIGHT")
	width, _ := strconv.Atoi(widthStr)
	height, _ := strconv.Atoi(heightStr)
	go sfu.Start(width, height, tailnet)
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
	if err := godotenv.Load(); err != nil {
		return
	}
	logger := InitLogger()
	defer logger.Sync()

	logger.Info("Starting SFU Server...")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Fatal("JWT_SECRET is not set in .env file")
		return
	}

	tailnet := os.Getenv("TAILNET")
	if tailnet == "" {
		logger.Fatal("TAILNET is not set in .env file")
		return
	}

	startSignalServer(jwtSecret, logger)
	startSFU(logger, tailnet)

	startClient()
	
	logger.Info("SFU Server is running on port 8080")
	
	err := http.ListenAndServe(tailnet + ":8080", nil)
	if err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}


package signal

import "go.uber.org/zap"

var jwtSecret string
var logger *zap.Logger

var FromSFU = make(chan SFUMessage, 1000)
var ToSFU = make(chan SFUMessage, 1000)   

func SetJWTSecret(secret string) {
	jwtSecret = secret
}

func SetLogger(l *zap.Logger) {
	logger = l
}
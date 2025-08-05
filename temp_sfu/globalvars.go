package sfu

import (
	"webrtc-gateway/signal"

	"go.uber.org/zap"
)

var FromSignal chan signal.SFUMessage
var ToSignal chan signal.SFUMessage

var logger *zap.Logger

func SetLogger(l *zap.Logger) {
	logger = l
}
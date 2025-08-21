package sfu

import (
	"webrtc-gateway/signal"

	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

var FromSignal chan signal.SFUMessage
var ToSignal chan signal.SFUMessage

var height int
var width int

var logger *zap.Logger

var tailnet string

var se webrtc.SettingEngine

var me *webrtc.MediaEngine

var api *webrtc.API

func SetLogger(l *zap.Logger) {
	logger = l
}
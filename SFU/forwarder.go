package sfu

import "webrtc-gateway/signal"

func sendToClient(msg signal.SFUMessage) {
	ToSignal <- msg
}
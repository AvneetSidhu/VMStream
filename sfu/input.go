package sfu

import (
	"encoding/json"

	"github.com/go-vgo/robotgo"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

func parseInputMessage(data []byte) (InputMessage, error) {
	var message InputMessage
	if err := json.Unmarshal(data, &message); err != nil {
		logger.Error("Failed to parse input message", zap.Error(err))
		return InputMessage{}, err
	}
	logger.Debug("Parsed input message", zap.String("type", message.Type))
	return message, nil
}

func parsePayload(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		logger.Error("Failed to parse payload", zap.Error(err))
		return err
	}
	logger.Debug("Parsed payload", zap.Any("payload", v))
	return nil
}

func handleInput(dc *webrtc.DataChannel) {
	logger.Info("Handling input data channel", zap.String("label", dc.Label()))
	var message InputMessage
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		logger.Debug("Received message on input channel", zap.String("label", dc.Label()), zap.ByteString("data", msg.Data))

		message, _ = parseInputMessage(msg.Data)
		switch message.Type {
		case "key":
			var payload KeyboardInputPayload
			parsePayload(message.Payload, &payload)
			robotgo.KeyToggle(payload.Key, true)
			robotgo.KeyToggle(payload.Key, false)
		case "mouse-move":
			var payload MouseInputPayload
			parsePayload(message.Payload, &payload)
			robotgo.Move(payload.X, payload.Y)
		case "mouse-click-left":
			var payload MouseInputPayload
			parsePayload(message.Payload, &payload)
			robotgo.Click("left")
		case "mouse-click-right":
			var payload MouseInputPayload
			parsePayload(message.Payload, &payload)
			robotgo.Click("right")
		default:
			logger.Warn("Unknown input message type", zap.String("type", message.Type))
		}
	})

	dc.OnClose(func() {
		logger.Debug("Input data channel closed", zap.String("label", dc.Label()))
	})
}
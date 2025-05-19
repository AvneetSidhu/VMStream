package signal

var jwtSecret string
var pionName string

var fromPion = make(chan *Message) // channel for messages from pion
var toPion = make(chan *Message)   // channel for messages to pion

func SetJWTSecret(secret string) {
	jwtSecret = secret
}

func SetPionName(name string) {
	pionName = name
}
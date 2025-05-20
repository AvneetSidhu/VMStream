package signal

var jwtSecret string
var pionName string

var FromSFU = make(chan *Message, 1000) // channel for messages to be read from pion
var ToSFU = make(chan SFUMessage, 1000) // channel for messages to send to pion

func SetJWTSecret(secret string) {
	jwtSecret = secret
}

func SetPionName(name string) {
	pionName = name
}
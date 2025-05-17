package main

var fromPion = make(chan *Message) // channel for messages from pion
var toPion = make(chan *Message)   // channel for messages to pion
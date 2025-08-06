package sfu

import (
	"sync"
	"webrtc-gateway/signal"
)

var peers = make(map[string]Client)
var mu = sync.RWMutex{}
var broadcaster Broadcaster

func Start(w int, h int) {

	width = w
	height = h

	broadcaster = Broadcaster {
		mu: sync.RWMutex{},
		clients: make(map[string]*Client),
	}

	broadcaster.Start()

	for msg := range FromSignal {
		switch v := msg.(type) {
		case *signal.SFUOffer:
			handleOffer(v.GetClientID(), v.Payload.SDP)
		case *signal.SFUIceCandidate:
			handleICECandidate(v.GetClientID(), v.Payload)
		}
	}	
}

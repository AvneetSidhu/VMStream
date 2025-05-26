package sfu

import (
	"sync"
	"webrtc-gateway/signal"
)

var peers = make(map[string]Client)
var mu = sync.RWMutex{}

func Start() {
	for msg := range FromSignal {
		switch v := msg.(type) {
		case *signal.SFUOffer:
			// fmt.Println("Received Offer from client")
			handleOffer(v.GetClientID(), v.Payload.SDP)
		case *signal.SFUIceCandidate:
			handleICECandidate(v.GetClientID(), v.Payload)
		}
	}	
}

package sfu

import (
	"fmt"
	"sync"
	"webrtc-gateway/signal"

	"github.com/pion/webrtc/v3"
)

var peers = make(map[string]*webrtc.PeerConnection)
var mu = sync.RWMutex{}

func Start() {
	
	for msg := range FromSignal {
		// fmt.Println("message from: ", msg.GetClientID())
		switch v := msg.(type) {
		case *signal.SFUOffer:
			fmt.Println("Received Offer from client")
			handleOffer(v.GetClientID(), v.Payload.SDP)
		case *signal.SFUAnswer:
			fmt.Println("Answer SDP:", v.Payload.SDP)
		case *signal.SFUIceCandidate:
			handleICECandidate(v.GetClientID(), v.Payload)
		}
		
	}	
}

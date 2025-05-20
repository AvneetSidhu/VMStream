package sfu

import (
	"fmt"
	"webrtc-gateway/signal"
)

func Start() {

	for msg := range FromSignal {
		fmt.Println("message from: ", msg.GetClientID())
		switch v := msg.(type) {
		case *signal.SFUOffer:
			fmt.Println("SDP:", v.Payload.SDP)
		
		case *signal.SFUAnswer:
			fmt.Println("Answer SDP:", v.Payload.SDP)
		
		case *signal.SFUIceCandidate:
			fmt.Println("Candidate:", v.Payload.Candidate)
		}
		
	}	
}

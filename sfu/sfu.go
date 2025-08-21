package sfu

import (
	"sync"
	"webrtc-gateway/signal"

	"github.com/pion/webrtc/v3"
)

var peers = make(map[string]Client)
var mu = sync.RWMutex{}
var broadcaster Broadcaster

func Start(w int, h int, tailnet string) {
	me = &webrtc.MediaEngine{}
	me.RegisterDefaultCodecs()
	se = webrtc.SettingEngine{}
	se.SetNAT1To1IPs([]string{tailnet}, webrtc.ICECandidateTypeHost)
	api = webrtc.NewAPI(webrtc.WithSettingEngine(se), webrtc.WithMediaEngine(me))

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

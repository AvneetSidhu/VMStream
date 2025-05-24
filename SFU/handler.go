package sfu

import (
	"fmt"
	"webrtc-gateway/signal"

	"github.com/pion/webrtc/v3"
)

func cleanup(clientID string) {
	mu.Lock()
	if pc, ok := peers[clientID]; ok {
		_ = pc.Close()
		delete(peers, clientID)
		fmt.Println("Ended webRTC connection for: ", clientID)
	}
	mu.Unlock()
}

func handleOffer(clientID string, offerSDP string) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Println("Error Creating New Peer Connection for client: " + clientID)
		fmt.Println(err)
		return
	}

	pc.OnConnectionStateChange(func (state webrtc.PeerConnectionState){
		fmt.Println("state: ", state)
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
		 cleanup(clientID)
	 	}
	})

	mu.Lock()
	peers[clientID] = pc
	mu.Unlock()

	offer :=  webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP: offerSDP,
	}

	err = pc.SetRemoteDescription(offer)
	if err != nil {
		fmt.Println("Error Setting Remote Description for client: " + clientID)
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		fmt.Println("Error Creating Answer for client: " + clientID)
	}

	pc.SetLocalDescription(answer)

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil { 
			candidateJSON := c.ToJSON()
			// fmt.Println("local ice-candidate: ", candidateJSON)
			sendToClient( &signal.SFUIceCandidate {
				ClientID: clientID,
				Type: "ice-candidate",
				Payload: signal.IceCandidatePayload{
					Candidate: candidateJSON.Candidate,
					SDPMid: deref(candidateJSON.SDPMid),
					SDPMLineIndex: derefUint(candidateJSON.SDPMLineIndex),
				},
			})
		}
	})

	sendToClient( &signal.SFUAnswer {
		ClientID: clientID,
		Type: "answer",
		Payload: signal.AnswerPayload{
			SDP: answer.SDP,
		},
	})
}

func handleICECandidate(clientID string, payload signal.IceCandidatePayload) {
	mu.RLock()
	pc, ok := peers[clientID]
	mu.RUnlock()
	// fmt.Println("received remote ice-candidate: ", payload)
	if ok {
		pc.AddICECandidate(webrtc.ICECandidateInit{
			Candidate: payload.Candidate,
			SDPMid: &payload.SDPMid,
			SDPMLineIndex: &payload.SDPMLineIndex,
		})
	} else {
		fmt.Println("Error accesing peer connection: " + clientID)
	}
}
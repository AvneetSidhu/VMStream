package sfu

import (
	"fmt"
	"time"
	"webrtc-gateway/signal"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

func cleanup(clientID string) {
	mu.Lock()
	if pc, ok := peers[clientID]; ok {
		_ = pc.PeerConn.Close()
		delete(peers, clientID)
		fmt.Println("Ended webRTC connection for: ", clientID)
	}
	mu.Unlock()
}

func startStream(trackV *webrtc.TrackLocalStaticSample, trackA *webrtc.TrackLocalStaticSample){
	
	go func() {
		ticket := time.NewTicker(time.Second)
		for range ticket.C {
			err := trackV.WriteSample(media.Sample{
				Data: []byte{0x00},
				Duration: time.Second,
			})
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		for range ticker.C {
			err := trackA.WriteSample(media.Sample{
				Data: []byte{0xF8, 0xFF, 0xFE},
				Duration: 20 * time.Millisecond,
			})
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
}

func handleOffer(clientID string, offerSDP string) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	videoTrack, _ := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
		"video", "pion",
	)
	
	audioTrack, _ := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "pion",
	)

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Println("Error Creating New Peer Connection for client: " + clientID)
		fmt.Println(err)
		return
	}

	startStream(videoTrack, audioTrack)

	pc.OnConnectionStateChange(func (state webrtc.PeerConnectionState){
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			fmt.Println("WebRTC Connection Terminated for client: " + clientID)
			cleanup(clientID)
	 	}
	})

	mu.Lock()
	peers[clientID] = Client{
		ClientID: clientID,
		PeerConn: pc,
		VideoTrack: videoTrack,
		AudioTrack: audioTrack,
	}
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

	pc.AddTrack(audioTrack)
	pc.AddTrack(videoTrack)

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		fmt.Println("Error Creating Answer for client: " + clientID)
	}

	pc.SetLocalDescription(answer)

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil { 
			candidateJSON := c.ToJSON()
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
	client, ok := peers[clientID]
	mu.RUnlock()
	if ok {
		pc := client.PeerConn
		pc.AddICECandidate(webrtc.ICECandidateInit{
			Candidate: payload.Candidate,
			SDPMid: &payload.SDPMid,
			SDPMLineIndex: &payload.SDPMLineIndex,
		})
	} else {
		fmt.Println("Error accesing peer connection: " + clientID)
	}
}
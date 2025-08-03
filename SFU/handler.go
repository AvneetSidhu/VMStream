package sfu

import (
	"encoding/json"
	"fmt"
	"webrtc-gateway/signal"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

func webRTCConnectionCleanup(clientID string) {
	if client, ok := broadcaster.GetClient(clientID); ok {
		_ = client.PeerConn.Close()
		broadcaster.RemoveClient(clientID)
		fmt.Println("Ended webRTC connection for: ", clientID)
	}
}

func handleOffer(clientID string, offerSDP string) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun1.l.google.com:19302"},
			},
		},
	}

	videoTrack, _ := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
		"video", "pion",
	)
	
	audioTrack, _ := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "pion",
	)

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Println("Error Creating New Peer Connection for client: " + clientID)
		fmt.Println(err)
		return
	}

	pc.OnConnectionStateChange(func (state webrtc.PeerConnectionState){
		fmt.Println("connection status: ", state)
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			fmt.Println("WebRTC Connection Terminated for client: " + clientID)
			webRTCConnectionCleanup(clientID)
	 	}
	})

	err = pc.SetRemoteDescription(
		webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP: offerSDP,
	})

	if err != nil {
		fmt.Println("Error Setting Remote Description for client: " + clientID)
		return
	}

	audioSender, _ := pc.AddTrack(audioTrack)
	videoSender, _ := pc.AddTrack(videoTrack)

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Println("Data channel opened:", d.Label())
	})

	readRTCPFeedback(audioSender, "audio")
	readRTCPFeedback(videoSender, "video")

	broadcaster.AddClient(
		&Client{
		ClientID: clientID,
		PeerConn: pc,
		VideoTrack: videoTrack,
		AudioTrack: audioTrack,
		AudioSSRC: uint32(audioSender.GetParameters().Encodings[0].SSRC),
		VideoSSRC: uint32(videoSender.GetParameters().Encodings[0].SSRC),
		})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil { 
			candidateJSON := c.ToJSON()
			jsonStr, _ := json.Marshal(candidateJSON)
			if candidateJSON.SDPMid == nil || *candidateJSON.SDPMid == "" || candidateJSON.SDPMLineIndex == nil {
				fmt.Println("Skipping ICE candidate with missing or empty sdpMid / sdpMLineIndex:", string(jsonStr))
				return
			}
			
			// fmt.Println("Sending ICE candidate JSON:", string(jsonStr))
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

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		fmt.Println("Error Creating Answer for client: " + clientID)
	}
	pc.SetLocalDescription(answer)

	gatherComplete := webrtc.GatheringCompletePromise(pc)
	<-gatherComplete

	finalAnswer := pc.LocalDescription()

	sendToClient( &signal.SFUAnswer {
		ClientID: clientID,
		Type: "answer",
		Payload: signal.AnswerPayload{
			SDP: finalAnswer.SDP,
		},
	})
}

func readRTCPFeedback(sender *webrtc.RTPSender, label string) {
    go func() {
        buf := make([]byte, 1500)
        for {
            n, _, err := sender.Read(buf)
            if err != nil {
                fmt.Printf("RTCP read error (%s): %v\n", label, err)
                return
            }

            pkts, err := rtcp.Unmarshal(buf[:n])
            if err != nil {
                fmt.Printf("Failed to parse RTCP (%s): %v\n", label, err)
                continue
            }

            for _, pkt := range pkts {
                switch p := pkt.(type) {
                case *rtcp.ReceiverReport:
                    for _, rr := range p.Reports {
                        fmt.Printf("[%s] RTCP RR: SSRC=%d, Lost=%d, Jitter=%d, RTT=%d\n",
                            label, rr.SSRC, rr.TotalLost, rr.Jitter, rr.LastSenderReport)
                        // You can use these stats to adjust bitrate or log performance
                    }
                case *rtcp.TransportLayerNack:
                    fmt.Printf("[%s] RTCP NACK: %+v\n", label, p)
                    // Optional: implement retransmission handling
                case *rtcp.PictureLossIndication:
                    fmt.Printf("[%s] RTCP PLI: SSRC=%d\n", label, p.MediaSSRC)
                    // Optional: trigger keyframe from encoder
                default:
                    fmt.Printf("[%s] RTCP: Unhandled packet type %T\n", label, pkt)
                }
            }
        }
    }()
}


func handleICECandidate(clientID string, payload signal.IceCandidatePayload) {
	client, ok := broadcaster.GetClient(clientID)
	if ok {
		pc := client.PeerConn
		// fmt.Println("received from client:", payload)
		pc.AddICECandidate(webrtc.ICECandidateInit{
			Candidate: payload.Candidate,
			SDPMid: &payload.SDPMid,
			SDPMLineIndex: &payload.SDPMLineIndex,
		})
	} else {
		fmt.Println("Error accesing peer connection: " + clientID)
	}
}
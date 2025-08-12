package sfu

import (
	"encoding/json"
	"fmt"
	"webrtc-gateway/signal"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

func webRTCConnectionCleanup(clientID string) {
	if client, ok := broadcaster.GetClient(clientID); ok {
		_ = client.PeerConn.Close()
		broadcaster.RemoveClient(clientID)
		logger.Info("Ended webRTC connection for: " + clientID)
	}
}

func handleOffer(clientID string, offerSDP string) {
	var dataChannel *webrtc.DataChannel
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
		logger.Error("Error Creating New Peer Connection", zap.String("clientID", clientID), zap.Error(err))
		return
	}

	pc.OnConnectionStateChange(func (state webrtc.PeerConnectionState){
		logger.Info("WebRTC Connection State has changed to " + state.String() + " for client: " + clientID)
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			logger.Info("WebRTC Connection Terminated for client: " + clientID)
			webRTCConnectionCleanup(clientID)
	 	}
	})

	err = pc.SetRemoteDescription(
		webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP: offerSDP,
	})

	if err != nil {
		logger.Error("Error Setting Remote Description", zap.String("clientID", clientID), zap.Error(err))
		return
	}

	audioSender, _ := pc.AddTrack(audioTrack)
	videoSender, _ := pc.AddTrack(videoTrack)

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		logger.Info("Data channel opened", zap.String("label", dc.Label()), zap.String("clientID", clientID))
		if dc.Label() == "input" {
			clients := broadcaster.GetAllClientIDs()

			payload, err := json.Marshal(ClientListPayload{
				Clients: clients,
			})

			if err != nil {
				logger.Error("Error marshalling client list payload", zap.Error(err))
			}

			broadcaster.SendMessage(dc,
				OutgoingMessage {
					Type: "viewer-list",
					Payload: json.RawMessage(payload),
				},
			)

				dataChannel = dc
				go handleInput(dc)
		}
	})

	readRTCPFeedback(audioSender, "audio")
	readRTCPFeedback(videoSender, "video")

	broadcaster.AddClient(
		&Client{
		ClientID: clientID,
		PeerConn: pc,
		VideoTrack: videoTrack,
		AudioTrack: audioTrack,
		dataChan: dataChannel,
		AudioSSRC: uint32(audioSender.GetParameters().Encodings[0].SSRC),
		VideoSSRC: uint32(videoSender.GetParameters().Encodings[0].SSRC),
		})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil { 
			candidateJSON := c.ToJSON()
			jsonStr, _ := json.Marshal(candidateJSON)
			if candidateJSON.SDPMid == nil || *candidateJSON.SDPMid == "" || candidateJSON.SDPMLineIndex == nil {
				logger.Warn("Skipping ICE candidate with missing or empty sdpMid / sdpMLineIndex", zap.String("candidate", string(jsonStr)))
				return
			}
			
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
		logger.Error("Error Creating Answer", zap.String("clientID", clientID), zap.Error(err))
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
				logger.Error("RTCP read error", zap.String("label", label), zap.Error(err))
				return
			}

			pkts, err := rtcp.Unmarshal(buf[:n])
			if err != nil {
				logger.Error("RTCP unmarshal error", zap.String("label", label), zap.Error(err))
				continue
			}

			for _, pkt := range pkts {
				switch p := pkt.(type) {
				case *rtcp.ReceiverReport:
					for _, rr := range p.Reports {
						logger.Debug("RTCP Receiver Report",
							zap.String("label", label),
							zap.Uint32("ssrc", rr.SSRC),
							zap.Uint32("lost", rr.TotalLost),
							zap.Uint32("jitter", rr.Jitter),
							zap.Uint32("rtt", rr.LastSenderReport),
						)
					}
				case *rtcp.TransportLayerNack:
					logger.Debug("RTCP NACK received",
						zap.String("label", label),
						zap.Any("nack", p),
					)
				case *rtcp.PictureLossIndication:
					logger.Debug("RTCP PLI received",
						zap.String("label", label),
						zap.Uint32("media_ssrc", p.MediaSSRC),
					)
				default:
					logger.Debug("Unhandled RTCP packet type",
						zap.String("label", label),
						zap.String("type", fmt.Sprintf("%T", pkt)),
					)
				}
			}
		}
	}()
}



func handleICECandidate(clientID string, payload signal.IceCandidatePayload) {
	client, ok := broadcaster.GetClient(clientID)
	if ok {
		pc := client.PeerConn
		pc.AddICECandidate(webrtc.ICECandidateInit{
			Candidate: payload.Candidate,
			SDPMid: &payload.SDPMid,
			SDPMLineIndex: &payload.SDPMLineIndex,
		})
	} else {
		logger.Error("Error accessing peer connection", zap.String("clientID", clientID))
	}
}
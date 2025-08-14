package sfu

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"sync/atomic"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"go.uber.org/zap"
)

type Client struct {
	ClientID         string
	PeerConn   *webrtc.PeerConnection
	VideoTrack *webrtc.TrackLocalStaticRTP
	AudioTrack *webrtc.TrackLocalStaticRTP

	audioChan chan *rtp.Packet
	videoChan chan *rtp.Packet
	dataChan *webrtc.DataChannel

	masterRTPStartTime uint32 //audio
	masterWallClockTime time.Time

	nonMasterRTPStartTime uint32 //video
	nonMasterWallClockTime time.Time

	AudioPacketsSent uint32
	AudioBytesSent   uint32

	VideoPacketsSent uint32
	VideoBytesSent   uint32

	LatestAudioPacketTime uint32
	LatestVideoPacketTime uint32

	AudioSSRC uint32
	VideoSSRC uint32

	done chan struct{}
}

type Broadcaster struct {
	mu sync.RWMutex
	clients map[string]*Client
	currentController string
}

type InputMessage struct {
	Type string `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type ControllerUpdatePayload struct {
	ClientID string `json:"clientID"`
}

type KeyboardInputPayload struct {
	Key string `json:"key"`
	Action string `json:"action"`
}

type MouseInputPayload struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Action string `json:"action"`
}

type OutgoingMessage struct {
	Type string `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type ClientListPayload struct {
	Clients []string `json:"clients"`
}

func (c *Client) sendRTCPSenderReport(ssrc uint32, rtpTimeStamp uint32, packetCount uint32, byteCount uint32) {
	now := time.Now()
	ntpTimeStamp := toNTPTime(now)

	senderReport := &rtcp.SenderReport{
		SSRC: ssrc,
		NTPTime: ntpTimeStamp,
		RTPTime: rtpTimeStamp,
		PacketCount: packetCount,
		OctetCount: byteCount,
	}

	_ = c.PeerConn.WriteRTCP([]rtcp.Packet{senderReport})
}

func (b *Broadcaster) AddClient(client *Client) {
	client.audioChan = make(chan *rtp.Packet, 1000)
	client.videoChan = make(chan *rtp.Packet, 1000)

	client.done = make(chan struct{})

	go func() {
		const audioClockRate = 48000
		for {
			select {
			case pkt, ok := <-client.audioChan:
				if !ok {
					return
				}

				delta := pkt.Timestamp - client.masterRTPStartTime
				targetTime := client.masterWallClockTime.Add(
					time.Duration(delta) * time.Second / audioClockRate,
				)

				sleep := time.Until(targetTime)
				if sleep > 0 {
					time.Sleep(sleep)
				}

				_ = client.AudioTrack.WriteRTP(pkt)
				atomic.StoreUint32(&client.LatestAudioPacketTime, pkt.Timestamp)

			case <-client.done:
				return
			}
		}
	}()

	go func() {
	const videoClockRate = 90000
	for {
		select {
		case pkt, ok := <-client.videoChan:
			if !ok {
				return
			}

			delta := pkt.Timestamp - client.nonMasterRTPStartTime
			targetTime := client.nonMasterWallClockTime.Add(
				time.Duration(delta) * time.Second / videoClockRate,
			)

			sleep := time.Until(targetTime)
			if sleep > 0 {
				time.Sleep(sleep)
			}

			_ = client.VideoTrack.WriteRTP(pkt)
			atomic.StoreUint32(&client.LatestVideoPacketTime, pkt.Timestamp)

		case <-client.done:
			return
		}
	}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				client.sendRTCPSenderReport(client.VideoSSRC, client.nonMasterRTPStartTime, client.VideoPacketsSent, client.VideoBytesSent)
				client.sendRTCPSenderReport(client.AudioSSRC, client.masterRTPStartTime, client.AudioPacketsSent, client.AudioBytesSent)
			case <-client.done:
				return
			}
		}

	}()

	b.mu.Lock() 
	b.clients[client.ClientID] = client
	b.mu.Unlock()
}

func (b *Broadcaster) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()

	go b.ingestRTP(5004, "video")
	go b.ingestRTP(5006, "audio")
}

func (b *Broadcaster) SetController(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.currentController = clientID
	logger.Info("Set current controller", zap.String("clientID", clientID))
}

func (b *Broadcaster) updateController(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.currentController = clientID
	logger.Info("Updated current controller", zap.String("clientID", clientID))
}

func (b *Broadcaster) ingestRTP(port int, mediaType string) {
	conn, err := net.ListenPacket("udp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		logger.Fatal("Failed to listen on port", zap.Int("port", port), zap.Error(err))
		return
	}
	defer conn.Close()
	logger.Info("Listening on port", zap.Int("port", port), zap.String("mediaType", mediaType))
	buf := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			logger.Error("Read error", zap.Error(err))
			continue
		}
		packet := make([]byte, n)
		copy(packet, buf[:n])
		b.forwardRTP(packet, mediaType)
	}
}

func (b *Broadcaster) RemoveClient(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	client, ok := b.clients[clientID]
	if ok {
		close(client.done)
		close(client.audioChan)
		close(client.videoChan)
		delete(b.clients, clientID)
	}
}

func (b *Broadcaster) GetClient(clientID string) (*Client, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	client, exists := b.clients[clientID]
	return client, exists
}

func (b *Broadcaster) GetAllClients() map[string]*Client {
	b.mu.RLock()
	defer b.mu.RUnlock()
	clientsCopy := make(map[string]*Client)
	for k, v := range b.clients {
		clientsCopy[k] = v
	}
	return clientsCopy
}

func (b *Broadcaster) GetAllClientIDs() []string {
	clients := broadcaster.GetAllClients()
	clientIDs := make([]string, 0, len(clients))
	for clientID := range clients {
		clientIDs = append(clientIDs, clientID)
	}
	return clientIDs
}

func (b *Broadcaster) SendMessage(dc *webrtc.DataChannel, message OutgoingMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		logger.Error("Failed to marshal outgoing message", zap.Error(err))
		return
	}
	dc.SendText(string(messageBytes))
}

func (b *Broadcaster) SendClientList() {
	clientIDs := b.GetAllClientIDs()
	payload, err := json.Marshal(ClientListPayload{
			Clients: clientIDs,
		})

	if err != nil {
		logger.Error("Error marshalling client list payload", zap.Error(err))
	}

	message :=	OutgoingMessage {
		Type: "viewer-list",
		Payload: json.RawMessage(payload),
		}
	
	clients := b.GetAllClients()
	for _, client := range clients {
		dc := client.dataChan
		if dc != nil && dc.ReadyState() == webrtc.DataChannelStateOpen {
			b.SendMessage(dc, message)
		}
	}
}

func (b *Broadcaster) forwardRTP(packet []byte, mediaType string) {
	clients := b.GetAllClients()

	rtpPacket := &rtp.Packet{}
	if err := rtpPacket.Unmarshal(packet); err != nil {
		logger.Error("Failed to unmarshal RTP packet", zap.Error(err))
		return
	}

	for _, client := range clients {
		packetCopy := *rtpPacket
		packetCopy.Payload = append([]byte(nil), rtpPacket.Payload...) // deep copy
		payloadSize := uint32(len(packetCopy.Payload))

		switch mediaType {
		case "audio":
			if client.masterRTPStartTime == 0 {
				client.masterRTPStartTime = packetCopy.Timestamp
				client.masterWallClockTime = time.Now()
			}
			client.AudioPacketsSent++
			client.AudioBytesSent += payloadSize

			select {
			case client.audioChan <- &packetCopy:
			default:
				select {
				case <-client.audioChan: // Drop oldest
				default:
				}
				select {
				case client.audioChan <- &packetCopy:
				default:
					logger.Warn("Client audio channel is full, dropping audio packet", zap.String("clientID", client.ClientID))
				}
			}

		case "video":
			if client.nonMasterRTPStartTime == 0 {
				client.nonMasterRTPStartTime = packetCopy.Timestamp
				client.nonMasterWallClockTime = time.Now()
			}
			client.VideoPacketsSent++
			client.VideoBytesSent += payloadSize

			select {
			case client.videoChan <- &packetCopy:
			default:
				select {
				case <-client.videoChan: // Drop oldest
				default:
				}
				select {
				case client.videoChan <- &packetCopy:
				default:
					logger.Warn("Client video channel is full, dropping video packet", zap.String("clientID", client.ClientID))
				}
			}
		}
	}
}

func (b *Broadcaster) GetCurrentController() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.currentController
}

func (b *Broadcaster) handleInput(dc *webrtc.DataChannel, clientID string) {
	logger.Info("Handling input data channel", zap.String("label", dc.Label()))
	var message InputMessage
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		logger.Debug("Received message on input channel", zap.String("label", dc.Label()), zap.ByteString("data", msg.Data))

		message, _ = parseInputMessage(msg.Data)
		switch message.Type {
		case "control":
			var payload ControllerUpdatePayload
			parsePayload(message.Payload, &payload)
			b.updateController(payload.ClientID)
		case "key":
			if b.GetCurrentController() != clientID {
				return
			}
			var payload KeyboardInputPayload
			parsePayload(message.Payload, &payload)
			handleKeyboardInput(payload.Key)
		case "mouse":
			if b.GetCurrentController() != clientID {
				return
			}
			var payload MouseInputPayload
			parsePayload(message.Payload, &payload)
			handleMouseInput(payload.X, payload.Y, payload.Action)
		default:
			logger.Warn("Unknown input message type", zap.String("type", message.Type))
		}
	})

	dc.OnClose(func() {
		logger.Debug("Input data channel closed", zap.String("label", dc.Label()))
	})
}





package sfu

import (
	"fmt"
	"net"
	"sync"
	"time"

	"sync/atomic"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	ClientID         string
	PeerConn   *webrtc.PeerConnection
	VideoTrack *webrtc.TrackLocalStaticRTP
	AudioTrack *webrtc.TrackLocalStaticRTP

	audioChan chan *rtp.Packet
	videoChan chan *rtp.Packet

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

type Broadcaster struct {
	mu sync.RWMutex
	clients map[string]*Client
}

func (b *Broadcaster) AddClient(client *Client) {
	
	client.audioChan = make(chan *rtp.Packet, 1000)
	client.videoChan = make(chan *rtp.Packet, 1000)

	client.done = make(chan struct{})
	
	// go func() {
	// 	for {
	// 		select {
	// 		case pkt, ok := <-client.audioChan:
	// 			if !ok {
	// 				return // channel closed
	// 			}
	// 			_ = client.AudioTrack.WriteRTP(pkt)
	// 		case <-client.done:
	// 			return
	// 		}
	// 	}
	// }()

	// go func() {
	// 	for {
	// 		select {
	// 		case pkt, ok := <-client.videoChan:
	// 			if !ok {
	// 				return
	// 			}
	// 			_ = client.VideoTrack.WriteRTP(pkt)
	// 		case <-client.done:
	// 			return
	// 		}
	// 	}
	// }()

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

func (b *Broadcaster) ingestRTP(port int, mediaType string) {
	conn, err := net.ListenPacket("udp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		fmt.Printf("Failed to listen on %d: %v", port , err)
	}
	defer conn.Close()
	fmt.Println("Listening on port:", port, "for ", mediaType)
	buf := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			fmt.Println("Read error:", err)
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

func (b *Broadcaster) forwardRTP(packet []byte, mediaType string) {
	clients := b.GetAllClients()

	rtpPacket := &rtp.Packet{}
	if err := rtpPacket.Unmarshal(packet); err != nil {
		fmt.Println("Failed to unmarshal RTP:", err)
		return
	}

	for _, client := range clients {
		packetCopy := *rtpPacket
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
				case <-client.audioChan: // drop the oldest packet and try to send the new one
				default:
					fmt.Printf("Client %s audio channel is full, dropping packet\n", client.ClientID)
				}

				select {
				case client.audioChan <- &packetCopy:
				default:
					fmt.Printf("Client %s audio channel is full, dropping packet\n", client.ClientID)
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
				case <-client.videoChan: // drop the oldest packet and try to send the new one
				default:
					fmt.Printf("Client %s video channel is full, dropping packet\n", client.ClientID)
				}

				select {
				case client.videoChan <- &packetCopy:
				default:
					fmt.Printf("Client %s video channel is full, dropping packet\n", client.ClientID)
				}
			}
		}
	}
}




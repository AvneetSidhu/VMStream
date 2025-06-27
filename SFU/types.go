package sfu

import (
	"fmt"
	"net"
	"sync"
	"time"

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

	AudioSSRC uint32
	VideoSSRC uint32

	AudioBindTime time.Time
	VideoBindTime time.Time

	done chan struct{}
}

type Broadcaster struct {
	mu sync.RWMutex
	clients map[string]*Client
}

func (b *Broadcaster) AddClient(client *Client) {
	
	client.audioChan = make(chan *rtp.Packet, 500)
	client.videoChan = make(chan *rtp.Packet, 500)
	client.done = make(chan struct{})
	
	go func() {
		for {
			select {
			case pkt, ok := <-client.audioChan:
				if !ok {
					return // channel closed
				}
				_ = client.AudioTrack.WriteRTP(pkt)
			case <-client.done:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case pkt, ok := <-client.videoChan:
				if !ok {
					return
				}
				_ = client.VideoTrack.WriteRTP(pkt)
			case <-client.done:
				return
			}
		}
	}()

	go func(c *Client) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()

			if c.AudioSSRC != 0 {
				audioSR := &rtcp.SenderReport{
					SSRC:    c.AudioSSRC,
					NTPTime: toNTPTime(now),
					RTPTime: uint32(now.Sub(c.AudioBindTime).Seconds() * 48000),
				}
				if err := c.PeerConn.WriteRTCP([]rtcp.Packet{audioSR}); err != nil {
					fmt.Printf("❌ Failed to write audio SR for client %s: %v\n", c.ClientID, err)
				}
			}

			if c.VideoSSRC != 0 {
				videoSR := &rtcp.SenderReport{
					SSRC:    c.VideoSSRC,
					NTPTime: toNTPTime(now),
					RTPTime: uint32(now.Sub(c.VideoBindTime).Seconds() * 90000),
				}
				if err := c.PeerConn.WriteRTCP([]rtcp.Packet{videoSR}); err != nil {
					fmt.Printf("❌ Failed to write video SR for client %s: %v\n", c.ClientID, err)
				}
			}

			fmt.Println("✅ Sent RTCP Sender Report for client:", c.ClientID)
		}
	}(client)

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

func startRTCPTicker(b *Broadcaster) {

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
		// fmt.Println("Received an", mediaType, "Packet")
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

func (b *Broadcaster) forwardRTP (packet []byte, mediaType string) {

	clients := b.GetAllClients()

	rtpPacket := &rtp.Packet{}
	if err := rtpPacket.Unmarshal(packet); err != nil {
		fmt.Println("Failed to unmarshal RTP:", err)
		return
	}

	for _, client := range clients {
		packetCopy := *rtpPacket

		switch mediaType {
		case "audio":
			// Set SSRC and bind time if not already set
			if client.AudioSSRC == 0 {
				client.AudioSSRC = packetCopy.SSRC
				client.AudioBindTime = time.Now()
			}
			select {
			case client.audioChan <- &packetCopy:
			default:
				fmt.Printf("Client %s audio channel is full, dropping packet\n", client.ClientID)
			}
		case "video":
			if client.VideoSSRC == 0 {
				client.VideoSSRC = packetCopy.SSRC
				client.VideoBindTime = time.Now()
			}
			select {
			case client.videoChan <- &packetCopy:
			default:
				fmt.Printf("Client %s video channel is full, dropping packet\n", client.ClientID)
			}
		}
	}
}

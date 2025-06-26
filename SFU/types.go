package sfu

import (
	"fmt"
	"net"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	ClientID         string
	PeerConn   *webrtc.PeerConnection
	VideoTrack *webrtc.TrackLocalStaticRTP
	AudioTrack *webrtc.TrackLocalStaticRTP
}

type Broadcaster struct {
	mu sync.RWMutex
	clients map[string]*Client
}

func (b *Broadcaster) AddClient(client *Client) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[client.ClientID] = client
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
		// fmt.Println("Received an", mediaType, "Packet")
		packet := make([]byte, n)
		copy(packet, buf[:n])
		b.forwardRTP(packet, mediaType)
	}
}

func (b *Broadcaster) RemoveClient(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, clientID)
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

	for _, client := range clients{
		packetCopy := *rtpPacket
		var err error
		if mediaType == "video" && client.VideoTrack != nil {
			err = client.VideoTrack.WriteRTP(&packetCopy)
		} else if mediaType == "audio" && client.AudioTrack != nil {
			err = client.AudioTrack.WriteRTP(&packetCopy)
		}

		if err != nil {
			fmt.Printf("Failed to send %s to client %s: %v", mediaType, client.ClientID, err)
			break
		}
	}
}


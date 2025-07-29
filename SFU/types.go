package sfu

import (
	"fmt"
	"net"
	"sync"
	"time"

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

	audioBuffer *RingBuffer
	videoBuffer *RingBuffer 

	masterRTPStartTime uint32
	masterWallClockTime time.Time

	nonMasterRTPStartTime uint32
	nonMasterWallClockTime time.Time

	AudioSSRC uint32
	VideoSSRC uint32

	AudioBindTime time.Time
	VideoBindTime time.Time

	AudioPacketsSent uint32
	AudioBytesSent   uint32

	VideoPacketsSent uint32
	VideoBytesSent   uint32

	done chan struct{}
}

type Broadcaster struct {
	mu sync.RWMutex
	clients map[string]*Client
}

func (b *Broadcaster) AddClient(client *Client) {
	
	client.audioChan = make(chan *rtp.Packet, 150)
	client.videoChan = make(chan *rtp.Packet, 150)

	client.audioBuffer = NewRingBuffer(30)
	client.videoBuffer = NewRingBuffer(30)

	client.done = make(chan struct{})
	
	go func() {
		for {
			select {
			case pkt, ok := <-client.audioChan:
				if !ok {
					return // channel closed
				}
				// _ = client.AudioTrack.WriteRTP(pkt)
				client.audioBuffer.Push(pkt)
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
				// _ = client.VideoTrack.WriteRTP(pkt)
				client.videoBuffer.Push(pkt)
			case <-client.done:
				return
			}
		}
	}()

	go func() {
		for {
			audioPkt, errA := client.audioBuffer.Pop()
			videoPkt, errV := client.videoBuffer.Pop()

			if errA != nil || errV != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			deltaA := audioPkt.Timestamp - client.masterRTPStartTime
			deltaV := videoPkt.Timestamp - client.nonMasterRTPStartTime

			playTimeA := client.masterWallClockTime.Add(time.Duration(deltaA) * time.Second / 48000)
			playTimeV := client.nonMasterWallClockTime.Add(time.Duration(deltaV) * time.Second / 90000)

			avDiff := absTimeDiff(playTimeA, playTimeV)

			if avDiff < 25 * time.Millisecond {
				client.AudioTrack.WriteRTP(audioPkt)
				client.VideoTrack.WriteRTP(videoPkt)
				continue
			}

			if playTimeA.Before(playTimeV) {
				sleepDuration := time.Until(playTimeV)
				if sleepDuration > 0 {
					time.Sleep(sleepDuration)
				}
				client.AudioTrack.WriteRTP(audioPkt)
			} else {
				sleepDuration := time.Until(playTimeA)
				if sleepDuration > 0 {
					time.Sleep(sleepDuration)
				}
				client.VideoTrack.WriteRTP(videoPkt)
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
				fmt.Printf("Client %s audio channel is full, dropping packet\n", client.ClientID)
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
				fmt.Printf("Client %s video channel is full, dropping packet\n", client.ClientID)
			}
		}
	}
}

type RingBuffer struct {
	buf []*rtp.Packet
	size int
	head int
	readPos int 
	writePos int
	count int
	mu sync.Mutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		buf: make([]*rtp.Packet, size),
		size: size,
	}
}

func (r *RingBuffer) Push(pkt *rtp.Packet) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == r.size {
		return
	}

	r.count++
	r.buf[r.writePos] = pkt
	r.writePos = (r.writePos + 1) % r.size
}

func (r *RingBuffer) Pop() (*rtp.Packet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == 0 {
		return nil, fmt.Errorf("ring buffer is empty")
	}

	pkt := r.buf[r.readPos]
	r.buf[r.readPos] = nil // Clear the slot
	r.readPos = (r.readPos + 1) % r.size
	r.count--

	return pkt, nil
}

func (r *RingBuffer) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}

func (r *RingBuffer) Capacity() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.size
}



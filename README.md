# VMStream

VMStream is a self-hosted, real-time VM streaming platform with authentication and multi-peer capabilities. It was written as an alternative to **HyperBeam**, a platform that offers embedded virtual browsers over the web. Unlike HyperBeam, VMStream gives me a persistent remote desktop that works like a personal computer. Instead of being locked to a new browser session every time, I can use any application that will run on the OS installed on my VM, keep files and history and pick up where I left off. Since the vm runs on my infrastructure, I own the data, preserve my state, and avoid the privacy concerns that come with third-party services.




## Usage:

Clone the 'host-frontend' branch of the repo into the vm you want to stream and run: `go build` in the root of the project to compile. The resulting executable is the server that will run inside the virtual machine that will be streamed. This branch contains a frontend that is hosted by the sfu server itself. Usage will also require the vm to have its own ip address / exposure to the private network. My current set up uses a tailnet and ACLs to expose the VM to users I want to grant access to.

You will also need a way to provide audio and video packets to the SFU to be forwarded. This can be done easily using GStreamer which will capture both audio and video and forward them as RTP. Forward Audio to `port: 5004` and video to `port: 5006`. Each individual packet size should be lower than 1280 bytes, as it 1280 is Tailscale's MTU size, any bigger and you will experience fragmentation and packet loss.

Ex: `gst-launch-1.0 -v rtpbin name=rtpbin \
  ximagesrc use-damage=0 ! video/x-raw,framerate=30/1 ! videoconvert ! videoscale ! video/x-raw,width=1280,height=720 ! queue ! \
    vp8enc deadline=1 cpu-used=1 target-bitrate=2500000 ! rtpvp8pay mtu=1200 ! rtpbin.send_rtp_sink_0 \
  rtpbin.send_rtp_src_0 ! udpsink host=127.0.0.1 port=5004 \
  pulsesrc ! audioresample ! audioconvert ! opusenc bitrate=128000 ! rtpopuspay mtu=1200 ! rtpbin.send_rtp_sink_1 \
  rtpbin.send_rtp_src_1 ! udpsink host=127.0.0.1 port=5006`

This command is capturing audio from PulseAudio and captures video from the display output of the VM.

You will also need the following environment variables:

| Variable    | Description                                      | Example / Notes            |
|------------|--------------------------------------------------|----------------------------|
| LOG_LEVEL  | Sets the logging verbosity (e.g., debug, info) | debug                     |
| JWT_SECRET | Secret key for signing JSON Web Tokens          | your-secret-key-here      |
| WIDTH      | Width of stream viewport             | 1280                      |
| HEIGHT     | Height of the stream viewport             | 720                       |
| TAILNET    | Tailnet IP of the machine to be streamed       | 100.xx.xxx.xxx

Lastly, to avoid having to port-forward or pay for hosting, you can run this application within a virtual machine that you can connect to using tailscale. Simply add the vm or device you plan to stream to a tailnet and connect via the ip address or magicDNS.

## How it's Made:

#### Tech Used: Go, WebRTC, SQLite3, GStreamer

This project is split into 2 packages: **sfu** and **signal**.

### Signal 
The signal package handles authentication, registration, database access, and the WebRTC handshake. It exposes API endpoints such as:
- `/login` - Authenticates users and issues an auth token
- `/register`- Creates a new user
- `/connect` - Opens a WebSocket used for exchanging SDP offers/answers and ICE candidates

When a client connects, it is added to an in-memory map of users that currently in the signaling process, and a dedicated read loop begins handling incoming messages from that user. Messages are parsed in to known types (Offer, ICE-Candidates and Termination), and are forwarded to the SFU via a Go channel. The sfu processes these messages and sends back responses to forward to clients using another go channel, which are routed to the correct client based on client ID. 

Passwords are hased using **bcrypt** and stored in an **SQLite** database. Once signaling is complete, the WebSocket is closed and the client continues their session through WebRTC.

### SFU

- The sfu package handles the core real-time media and input streaming. Its responsibilities include:

- Negotiating WebRTC connections

- Managing audio and video tracks (Opus for audio, VP8 for video)

- Ingesting RTP packets over UDP

- Forwarding media to clients

- Handling remote input injection

The SFU runs a central read loop to process signaling messages (Offers and ICE Candidates). ICE candidates are directly applied to peer connections, while Offers trigger the creation of a new peer connection with audio and video tracks. Offer handling is executed in a goroutine, since the process blocks until ICE gathering is complete. Once a connection is established, the client is added to the Broadcaster.

### The Broadcaster

At the heart of the SFU is the Broadcaster struct, which maintains session state. It:

- Tracks active clients in a client map

- Manages which client has control of the VM

- Handles RTP ingest and distribution

- Processes input events from clients

Audio and video are ingested on ports 5004 and 5006, respectively. GStreamer streams RTP packets over UDP into these ports, handling A/V synchronization upstream. VMStream ingests these RTP packets and forwards deep-copies to each connected client.

User inputs (keyboard, mouse, etc.) are sent as discrete events over a WebRTC data channel. Clients can also send “take control” requests via this channel. The Broadcaster verifies whether clients are currently the controller before injecting the input into the VM. The same data channel is used to broadcast an updated list of connected clients whenever someone joins or leaves the session.
## UI

The frontend is served by by VMStream from within the VM it is running on. This way I didn't have to worry about deploying the frontend and backend separately.


<img width="767" height="398" alt="image" src="https://github.com/user-attachments/assets/1a067066-bde5-4158-8065-56b356a0e915" />
<img width="767" height="398" alt="image" src="https://github.com/user-attachments/assets/7d5ea3b9-45ab-4d6b-88e4-9cd28d4689ba" />

## To-do

- [x]  Implement scrolling, click and drag, and other input types
- [ ]  CI/CD


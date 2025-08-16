
# VMStream

VMStream is a self-hosted, real-time VM streaming platform with authentication and multi-peer capabilities. It was written as an alternative to **HyperBeam**, a platform that offers embedded virtual browsers over the web. Unlike HyperBeam, VMStream gives me a persistent remote desktop that works like a personal computer. Instead of being locked to a new browser session every time, I can use any application that will run on the OS installed on my VM, keep files and history and pick up where I left off. Since the vm runs on my infrastructure, I own the data, preserve my state, and avoid the privacy concerns that come with third-party services.




## Usage:

Clone the repo and run: `go build` in the root of the project to compile. The resulting executable is the server that will run inside the virtual machine that will be streamed. 

You will also need a way to provide audio and video packets to the SFU to be forwarded. This can be done easily using GStreamer which will capture both audio and video and forward them as RTP. Forward Audio to `port: 5004` and video to `port: 5006`. 

Ex: `gst-launch-1.0 -v rtpbin name=rtpbin \
ximagesrc use-damage=0 ! video/x-raw,framerate=30/1 ! videoconvert ! queue ! vp8enc deadline=1 ! rtpvp8pay ! rtpbin.send_rtp_sink_0 \
rtpbin.send_rtp_src_0 ! udpsink host=127.0.0.1 port=5004 \
pulsesrc ! audioresample ! audioconvert ! opusenc ! rtpopuspay ! rtpbin.send_rtp_sink_1 \
rtpbin.send_rtp_src_1 ! udpsink host=127.0.0.1 port=5006`

This command is capturing audio from PulseAudio and captures video from the display output of the VM.

You will also need the following environment variables:

| Variable    | Description                                      | Example / Notes            |
|------------|--------------------------------------------------|----------------------------|
| LOG_LEVEL  | Sets the logging verbosity (e.g., debug, info) | debug                     |
| JWT_SECRET | Secret key for signing JSON Web Tokens          | your-secret-key-here      |
| WIDTH      | Width of the application viewport              | 1280                      |
| HEIGHT     | Height of the application viewport             | 720                       |


The http request bodies expected are of the form: 

**Login**

```json
{
    "username": "username",
    "password": "password",
    "auth": ""
}
```

**Register**

```json
{
    "username": "username",
    "password": "password",
    "auth": ""
}
```

**Connect**

*uses query string to pass auth

`/connect?client_id=${username}&auth=${auth}`

**WebSocket Signaling Messages**

```json
{
  "type": "offer",
  "clientId": "username",
  "payload": {
    "sdp": "your-sdp-string-here"
  }
}
```

```json
{
  "type": "ice-candidate",
  "clientId": "username",
  "payload": {
    "candidate": "candidate-string",
    "sdpMid": "0",
    "sdpMLineIndex": 0
  }
}
```

```json
{
  "type": "termination",
  "clientId": "username",
  "payload": {}
}
```
**Input Messages**

```json
{
  "type": "mouse",
  "payload": {
    "x": "normalizedX",
    "y": "normalizedY",
    "action": "move",
    "deltaY": ""
  }
}
```

```json
{
  "type": "mouse",
  "payload": {
    "x": "normalizedX",
    "y": "normalizedY",
    "action": "button",
    "deltaY": ""
  }
}
```

```json
{
  "type": "key",
  "payload": {
    "key": "event.key",
    "action": "keydown"
  }
}
```



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
## To-do

- [ ]  Fix SFU read loop bottleneck, introduce per-client reading from SFU
- [ ]  Decouple input injection from SFU, create a service on VM and move SFU outside
- [ ]  Introduce some sort of telemetry / performance measurement
- [ ]  Refactor json input commands to binary
- [ ]  CI/CD for VMStream and future Input-injection service

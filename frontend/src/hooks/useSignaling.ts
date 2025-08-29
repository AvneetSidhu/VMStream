import { useRef, useState } from "react";

export function useSignaling(username: string | null, auth: string | null) {
  if (!username) throw new Error("Username is required for signaling");
  if (!auth) throw new Error("Auth token is required for signaling");

  const pcRef = useRef<RTCPeerConnection | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const dataChannelRef = useRef<RTCDataChannel | null>(null);
  const [viewers, setViewers] = useState<string[]>([]);

  async function connect(videoRef: React.RefObject<HTMLVideoElement | null>) {
    if (!videoRef || !videoRef.current)
      throw new Error("Video ref is required");
    if (pcRef.current) pcRef.current.close();
    if (socketRef.current) socketRef.current.close();

    const pc = new RTCPeerConnection({
      iceServers: [{ urls: "stun:stun1.l.google.com:19302" }],
    });
    pcRef.current = pc;

    // Data channel
    const dc = pc.createDataChannel("input", { ordered: true });
    dataChannelRef.current = dc;

    dc.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg?.payload?.clients) setViewers(msg.payload.clients as string[]);
      } catch (e) {
        console.error("Bad DC message:", e);
      }
    };

    // ICE candidates
    pc.onicecandidate = (event) => {
      if (event.candidate && socketRef.current?.readyState === WebSocket.OPEN) {
        socketRef.current.send(
          JSON.stringify({
            type: "ice-candidate",
            clientId: username,
            payload: event.candidate,
          })
        );
      }
    };

    // Remote media
    pc.ontrack = (e) => {
      if (videoRef.current && !videoRef.current.srcObject) {
        videoRef.current.srcObject = e.streams[0];
      }
    };

    pc.addTransceiver("audio", { direction: "sendrecv" });
    pc.addTransceiver("video", { direction: "sendrecv" });

    // Offer
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);

    await new Promise<void>((resolve) => {
      if (pc.iceGatheringState === "complete") return resolve();
      const check = () => {
        if (pc.iceGatheringState === "complete") {
          pc.removeEventListener("icegatheringstatechange", check);
          resolve();
        }
      };
      pc.addEventListener("icegatheringstatechange", check);
    });

    // WebSocket
    const socket = new WebSocket(
      `api/connect?client_id=${username}&auth=${auth}`
    );
    socketRef.current = socket;

    socket.addEventListener("open", () => {
      socket.send(
        JSON.stringify({
          type: "offer",
          clientId: username,
          payload: { sdp: pc.localDescription?.sdp },
        })
      );
    });

    socket.addEventListener("message", async (event) => {
      const msg = JSON.parse(event.data);
      switch (msg.type) {
        case "answer":
          await pc.setRemoteDescription(
            new RTCSessionDescription({ type: "answer", sdp: msg.payload.sdp })
          );
          break;
        case "ice-candidate":
          if (msg.payload) {
            const cand = new RTCIceCandidate(msg.payload);
            await pc.addIceCandidate(cand).catch(console.error);
          }
          break;
      }
    });

    socket.addEventListener("close", () => {
      console.log("WebSocket closed");
    });
  }

  function disconnect() {
    if (pcRef.current) {
      pcRef.current.close();
      pcRef.current = null;
    }

    if (socketRef.current) {
      socketRef.current.close();
      socketRef.current = null;
    }

    dataChannelRef.current = null;
  }
  return {
    disconnect,
    viewers,
    connect,
    dataChannelRef,
    pcRef,
    socketRef,
  };
}

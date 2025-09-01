import React, { useEffect, useRef, useState } from "react";
import ViewerList from "./ViewerList";
import { useAuth } from "../authContext";
import { useSignaling } from "../hooks/useSignaling";

const Display: React.FC = () => {
  const { username, token: auth } = useAuth();
  const videoRef = useRef<HTMLVideoElement>(null);

  const { disconnect, viewers, connect, dataChannelRef } = useSignaling(
    username,
    auth
  );

  const [inputLocked, setInputLocked] = useState(false);
  const inputLockedRef = useRef(false);
  const listenersAttachedRef = useRef(false);
  var mouseDown = false;

  useEffect(() => {
    inputLockedRef.current = inputLocked;
  }, [inputLocked]);

  function handleMouseMove(event: MouseEvent) {
    if (inputLockedRef.current) return;
    const dc = dataChannelRef.current;
    const video = videoRef.current;
    if (!video || !dc || dc.readyState !== "open") return;

    const rect = video.getBoundingClientRect();
    const videoAspect = video.videoWidth / video.videoHeight;
    const elementAspect = rect.width / rect.height;

    let visibleWidth = rect.width;
    let visibleHeight = rect.height;
    let offsetX = 0;
    let offsetY = 0;

    if (elementAspect > videoAspect) {
      visibleWidth = rect.height * videoAspect;
      offsetX = (rect.width - visibleWidth) / 2;
    } else {
      visibleHeight = rect.width / videoAspect;
      offsetY = (rect.height - visibleHeight) / 2;
    }

    const normalizedX = (event.clientX - rect.left - offsetX) / visibleWidth;
    const normalizedY = (event.clientY - rect.top - offsetY) / visibleHeight;

    const clampedX = Math.min(Math.max(normalizedX, 0), 1);
    const clampedY = Math.min(Math.max(normalizedY, 0), 1);

    dc.send(
      JSON.stringify({
        type: "mouse",
        payload: {
          x: clampedX,
          y: clampedY,
          action: "move",
          dragging: mouseDown,
        },
      })
    );
  }

  function handleMouseDown(event: MouseEvent) {
    event.preventDefault();
    if (inputLockedRef.current) return;
    const dc = dataChannelRef.current;
    const video = videoRef.current;
    if (!video || !dc || dc.readyState !== "open") return;

    const rect = video.getBoundingClientRect();
    const normalizedX = (event.clientX - rect.left) / rect.width;
    const normalizedY = (event.clientY - rect.top) / rect.height;

    const button =
      event.button === 0
        ? "left-click"
        : event.button === 1
        ? "middle-click"
        : "right-click";
    console.log("mouse is down");
    mouseDown = true;
    dc.send(
      JSON.stringify({
        type: "mouse",
        payload: { x: normalizedX, y: normalizedY, action: button },
      })
    );
  }

  function handleMouseUp() {
    if (inputLockedRef.current) return;
    const dc = dataChannelRef.current;
    const video = videoRef.current;
    if (!video || !dataChannelRef || dc?.readyState !== "open") return;
    console.log("mouse is up");
    mouseDown = false;
  }

  function handleScroll(event: WheelEvent) {
    event.preventDefault();
    if (inputLockedRef.current) return;
    const dc = dataChannelRef.current;
    const video = videoRef.current;
    if (!video || !dc || dc.readyState !== "open") return;

    const dir = event.deltaY > 0 ? "down" : "up";

    dc.send(
      JSON.stringify({
        type: "scroll",
        payload: { direction: dir },
      })
    );
  }

  function handleKeyDown(event: KeyboardEvent) {
    if (inputLockedRef.current) return;
    const dc = dataChannelRef.current;
    if (!dc || dc.readyState !== "open") return;

    dc.send(
      JSON.stringify({
        type: "key",
        payload: { key: event.key, action: "keydown" },
      })
    );
  }

  function buttonClick() {
    if (!listenersAttachedRef.current && videoRef.current) {
      videoRef.current.addEventListener("mousemove", handleMouseMove);
      videoRef.current.addEventListener("mousedown", handleMouseDown);
      videoRef.current.addEventListener("mouseup", handleMouseUp);
      videoRef.current.addEventListener("wheel", handleScroll);
      window.addEventListener("keydown", handleKeyDown);
      listenersAttachedRef.current = true;
    }

    connect(videoRef);
  }

  function handleTakeMouse() {
    const dc = dataChannelRef.current;
    if (dc && dc.readyState === "open") {
      dc.send(
        JSON.stringify({ type: "control", payload: { clientId: username } })
      );
    }
  }

  function handleLockMouse() {
    setInputLocked((prev) => !prev);
  }

  function handleVolumeChange(e: React.ChangeEvent<HTMLInputElement>) {
    const video = videoRef.current;
    if (!video) return;
    const volume = parseFloat(e.target.value);
    if (video.muted && volume > 0) video.muted = false;
    video.volume = volume;
  }

  function handleFullScreen() {
    const container = document.getElementById("display-container");
    if (!container) return;

    if (document.fullscreenElement) {
      document.exitFullscreen();
    } else {
      container.requestFullscreen().catch((err) => {
        console.error("Failed to enter full screen:", err);
      });
    }
  }

  useEffect(() => {
    return () => {
      disconnect();
      const video = videoRef.current;
      if (video && listenersAttachedRef.current) {
        video.removeEventListener("mousemove", handleMouseMove);
        video.removeEventListener("mousedown", handleMouseDown);
        video.removeEventListener("wheel", handleScroll);
        listenersAttachedRef.current = false;
      }
      window.removeEventListener("keydown", handleKeyDown);
      if (video) video.srcObject = null;
    };
  }, []);

  return (
    <div
      id="display-container"
      className="bg-amber-950 flex flex-col w-full h-screen overflow-hidden"
    >
      <div className="flex flex-1 flex-row overflow-hidden">
        <div className="flex-1 bg-black flex">
          <video
            id="video"
            ref={videoRef}
            className="w-full h-full object-contain"
            autoPlay
            muted
            playsInline
          />
        </div>

        <div className="w-64 bg-gray-800 overflow-y-auto">
          <ViewerList viewers={viewers} />
        </div>
      </div>

      <div className="flex-shrink-0 display-controls w-full flex justify-between items-center p-2 bg-gray-700 text-white">
        <div className="flex space-x-2">
          <button
            className="px-3 py-1 bg-blue-600 rounded hover:bg-blue-700"
            onClick={buttonClick}
          >
            Connect
          </button>
        </div>

        <div className="flex items-center space-x-2">
          <button
            className="px-3 py-1 bg-blue-600 rounded hover:bg-blue-700"
            onClick={handleFullScreen}
          >
            Full Screen
          </button>
          <button
            className="px-3 py-1 bg-blue-600 rounded hover:bg-blue-700"
            onClick={handleTakeMouse}
          >
            Take Mouse
          </button>
          <button
            className="px-3 py-1 bg-blue-600 rounded hover:bg-blue-700"
            onClick={handleLockMouse}
          >
            {inputLocked ? "Unlock Mouse" : "Lock Mouse"}
          </button>
          <input
            type="range"
            min={0}
            max={1}
            step={0.01}
            defaultValue={0}
            onChange={handleVolumeChange}
            className="h-1 w-24 accent-blue-500 rounded"
          />
        </div>
      </div>
    </div>
  );
};

export default Display;

import { useEffect, useRef, useImperativeHandle, forwardRef, useCallback } from "react";
import { DvrPlayer as DvrPlayerCore } from "../lib/dvr-player";
import type { PlaybackOpenParams } from "../lib/dvr-protocol";

export interface DvrPlayerHandle {
  setSpeed: (speed: number) => void;
  seek: (timestampMs: number) => void;
  stop: () => void;
  captureFrame: (filename?: string) => void;
  getCanvas: () => HTMLCanvasElement | null;
}

interface DvrPlayerProps {
  cameraId: string;
  channelId: string;
  startTime: number; // Unix timestamp in seconds
  endTime: number;
  onTimeUpdate?: (timestamp: number) => void;
  onReady?: () => void;
  onError?: (msg: string) => void;
  onEnd?: () => void;
  className?: string;
}

const DvrPlayer = forwardRef<DvrPlayerHandle, DvrPlayerProps>(function DvrPlayer(
  { cameraId, channelId, startTime, endTime, onTimeUpdate, onReady, onError, onEnd, className },
  ref,
) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const playerRef = useRef<DvrPlayerCore | null>(null);

  // Stable callback refs to avoid re-creating player on callback changes
  const onTimeRef = useRef(onTimeUpdate);
  const onReadyRef = useRef(onReady);
  const onErrorRef = useRef(onError);
  const onEndRef = useRef(onEnd);
  onTimeRef.current = onTimeUpdate;
  onReadyRef.current = onReady;
  onErrorRef.current = onError;
  onEndRef.current = onEnd;

  useImperativeHandle(ref, () => ({
    setSpeed: (speed: number) => playerRef.current?.setSpeed(speed),
    seek: (timestampMs: number) => playerRef.current?.seek(timestampMs),
    stop: () => playerRef.current?.stop(),
    captureFrame: (filename?: string) => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      canvas.toBlob(
        (blob) => {
          if (!blob) return;
          const url = URL.createObjectURL(blob);
          const a = document.createElement("a");
          a.href = url;
          a.download = filename || `capture_${Date.now()}.jpg`;
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          URL.revokeObjectURL(url);
        },
        "image/jpeg",
        0.95,
      );
    },
    getCanvas: () => canvasRef.current,
  }));

  const startPlayback = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    // Destroy previous player if any
    playerRef.current?.destroy();

    const player = new DvrPlayerCore({
      canvas,
      cameraId,
      onTime: (ts) => onTimeRef.current?.(ts),
      onReady: () => onReadyRef.current?.(),
      onError: (msg) => onErrorRef.current?.(msg),
      onEnd: () => onEndRef.current?.(),
    });

    const params: PlaybackOpenParams = {
      channelId,
      startTime,
      endTime,
    };

    player.play(params);
    playerRef.current = player;
  }, [cameraId, channelId, startTime, endTime]);

  useEffect(() => {
    startPlayback();
    return () => {
      playerRef.current?.destroy();
      playerRef.current = null;
    };
  }, [startPlayback]);

  return (
    <canvas
      ref={canvasRef}
      className={className}
      style={{ display: "block", width: "100%", height: "100%", objectFit: "contain", background: "#000" }}
    />
  );
});

export default DvrPlayer;

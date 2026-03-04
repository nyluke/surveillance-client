import { useEffect, useRef, useImperativeHandle, forwardRef } from "react";
import { streamUrl } from "../lib/go2rtc";

export interface VideoPlayerHandle {
  captureFrame: (filename?: string) => void;
}

const VideoPlayer = forwardRef<
  VideoPlayerHandle,
  {
    cameraId: string;
    useSub?: boolean;
    className?: string;
  }
>(function VideoPlayer({ cameraId, useSub = false, className = "" }, ref) {
  const elRef = useRef<HTMLElement>(null);
  const src = streamUrl(cameraId, useSub);

  useImperativeHandle(ref, () => ({
    captureFrame: (filename?: string) => {
      const el = elRef.current as any;
      const video = el?.video as HTMLVideoElement | undefined;
      if (!video || video.videoWidth === 0) return;
      const canvas = document.createElement("canvas");
      canvas.width = video.videoWidth;
      canvas.height = video.videoHeight;
      const ctx = canvas.getContext("2d");
      if (!ctx) return;
      ctx.drawImage(video, 0, 0);
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
  }));

  useEffect(() => {
    const el = elRef.current;
    if (!el) return;
    (el as any).mode = "mse";
    (el as any).media = "video";
    (el as any).background = true;
    (el as any).visibilityThreshold = 0;
    (el as any).src = src;

    // video-rtc creates its <video> in connectedCallback (before useEffect),
    // so it already exists — disable the default controls
    const video = (el as any).video as HTMLVideoElement | undefined;
    if (video) video.controls = false;
  }, [src]);

  return (
    <video-rtc
      ref={elRef}
      className={className}
      style={{ display: "block", width: "100%", height: "100%" }}
    />
  );
});

export default VideoPlayer;

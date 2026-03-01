import { useEffect, useRef } from "react";
import { streamUrl } from "../lib/go2rtc";

export default function VideoPlayer({
  cameraId,
  useSub = false,
  className = "",
}: {
  cameraId: string;
  useSub?: boolean;
  className?: string;
}) {
  const ref = useRef<HTMLElement>(null);
  const src = streamUrl(cameraId, useSub);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    (el as any).mode = "mse";
    (el as any).background = true;
    (el as any).visibilityThreshold = 0;
    (el as any).src = src;
  }, [src]);

  return (
    <video-rtc
      ref={ref}
      className={className}
      style={{ display: "block", width: "100%", height: "100%" }}
    />
  );
}

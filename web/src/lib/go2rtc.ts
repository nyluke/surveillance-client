// Helper to generate go2rtc WebSocket URLs for video-rtc element

export function streamUrl(cameraId: string, sub = false): string {
  const name = sub ? `cam_${cameraId}_sub` : `cam_${cameraId}`;
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/go2rtc/api/ws?src=${name}`;
}

export function streamName(cameraId: string, sub = false): string {
  return sub ? `cam_${cameraId}_sub` : `cam_${cameraId}`;
}


// DVR WebSocket protocol — command builders ported from dvr-js/websocket.cmd.js

let requestId = 0;

function getBasic() {
  requestId = requestId < Number.MAX_SAFE_INTEGER ? requestId + 1 : 1;
  return {
    ver: "1.0",
    time: Date.now(),
    id: requestId,
    nonce: Math.floor(Math.random() * 2147483646 + 1),
  };
}

export function getRandomGUID(): string {
  let guid = "";
  const template = "00000000-0000-0000-0000-000000000000";
  const hex = "0123456789abcdef";
  for (let i = 0; i < 36; i++) {
    guid += template[i] === "-" ? "-" : hex[Math.floor(Math.random() * 16)];
  }
  return `{${guid}}`;
}

// All known recording event types (used as default type_mask)
export const REC_EVENT_TYPES = [
  "manual",
  "schedule",
  "motion",
  "sensor",
  "gsensor",
  "shelter",
  "overspeed",
  "overbound",
  "osc",
  "avd",
  "tripwire",
  "perimeter",
  "vfd",
  "pos",
  "smart_motion",
  "face_verity",
  "cpc",
  "cdd",
  "ipd",
  "smart_aoi_entry",
  "smart_aoi_leave",
  "smart_pass_line",
  "smart_plate_verity",
];

export interface PlaybackOpenParams {
  channelId: string; // DVR channel GUID e.g. "{00000001-0000-0000-0000-000000000000}"
  startTime: number; // Unix timestamp in seconds
  endTime: number; // Unix timestamp in seconds
  streamIndex?: number; // 0 = main, 1 = sub
  typeMask?: string[];
  backup?: boolean; // true for recording backup (raw H.264 → AVI)
  audio?: boolean; // true to include audio in backup
}

// Convert a 1-based channel number to DVR GUID format.
// chID=1 → "{00000001-0000-0000-0000-000000000000}"
// chID=11 → "{0000000B-0000-0000-0000-000000000000}"
export function channelToGuid(channelNumber: number): string {
  const hex = channelNumber.toString(16).toUpperCase().padStart(8, "0");
  return `{${hex}-0000-0000-0000-000000000000}`;
}

export interface DvrCommand {
  url: string;
  basic: { ver: string; time: number; id: number; nonce: number };
  data: Record<string, unknown>;
}

export function cmdPlaybackOpen(params: PlaybackOpenParams): DvrCommand {
  const taskId = getRandomGUID();
  const data: Record<string, unknown> = {
    task_id: taskId,
    channel_id: params.channelId,
    start_time: params.startTime,
    end_time: params.endTime,
    stream_index: params.streamIndex ?? 1,
    type_mask: params.typeMask ?? REC_EVENT_TYPES,
  };
  if (params.backup) data.backup = true;
  if (params.audio) data.audio = true;
  return {
    url: "/device/playback/open",
    basic: getBasic(),
    data,
  };
}

export function cmdPlaybackClose(taskId: string): DvrCommand {
  return {
    url: "/device/playback/close",
    basic: getBasic(),
    data: { task_id: taskId },
  };
}

export function cmdPlaybackSeek(taskId: string, frameTimeSec: number): DvrCommand {
  return {
    url: "/device/playback/seek",
    basic: getBasic(),
    data: { task_id: taskId, frame_time: frameTimeSec },
  };
}

export function cmdKeyFrame(taskId: string, frameTimeSec: number): DvrCommand {
  return {
    url: "/device/playback/key_frame",
    basic: getBasic(),
    data: { task_id: taskId, frame_time: frameTimeSec },
  };
}

export function cmdAllFrame(taskId: string, frameTimeSec: number): DvrCommand {
  return {
    url: "/device/playback/all_frame",
    basic: getBasic(),
    data: { task_id: taskId, frame_time: frameTimeSec },
  };
}

export function cmdRefreshFrameIndex(
  taskId: string,
  frameIndex: number,
): DvrCommand {
  return {
    url: "/device/playback/refresh_play_index",
    basic: getBasic(),
    data: { task_id: taskId, play_frame_index: frameIndex },
  };
}

// Convert a timestamp in milliseconds to Unix seconds
export function msToSeconds(timestampMs: number): number {
  return Math.floor(timestampMs / 1000);
}

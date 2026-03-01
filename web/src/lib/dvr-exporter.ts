// DVR recording export — opens a backup playback session, feeds binary data
// to the WASM decoder with onlyForRecord=true, and downloads the resulting AVI.
//
// Reverse-engineered from DVR's recordBuilder.js + websocket.recordBackup.js:
// - type: 0 (NOT 1) — critical for correct AVI muxing in WASM
// - onlyForRecord: true — raw compressed data goes directly to AVI, no decode
// - frameIndex callbacks from worker provide flow control indices
// - refreshFrameIndex sent every 8 frames to keep DVR sending data
// - backup: true in playback open tells DVR to send at disk speed (fast)

import {
  cmdPlaybackOpen,
  cmdPlaybackClose,
  cmdRefreshFrameIndex,
  type PlaybackOpenParams,
} from "./dvr-protocol";

export interface ExportOptions {
  cameraId: string;
  channelId: string; // DVR channel GUID
  startTime: number; // Unix seconds
  endTime: number; // Unix seconds
  onProgress?: (frameTimeMs: number) => void;
  onComplete?: (blob: Blob, filename: string) => void;
  onError?: (msg: string) => void;
}

export class DvrExporter {
  private ws: WebSocket | null = null;
  private worker: Worker | null = null;
  private taskId = "";
  private chunks: Uint8Array[] = [];
  private options: ExportOptions;
  private canRefreshIndex = false;
  private endHandled = false;
  private destroyed = false;

  // Diagnostics
  private binaryMsgCount = 0;
  private frameIndexCount = 0;
  private bufferErrorCount = 0;
  private totalRawBytes = 0;
  private startedAt = 0;

  constructor(options: ExportOptions) {
    this.options = options;
  }

  start(): void {
    this.startedAt = Date.now();
    this.worker = new Worker("/dvr-decoder/decoder.js");

    this.worker.onmessage = (ev) => {
      const msg = ev.data;
      if (!msg?.cmd || this.destroyed) return;

      switch (msg.cmd) {
        case "ready":
          // type: 0 — matches DVR's recordBuilder (live/preview mode).
          // type: 1 (playback) produces broken AVI files.
          // maxSingleSize: 0 → Infinity (no file splitting)
          console.log("[DvrExporter] worker ready, sending init type=0");
          this.worker!.postMessage({ cmd: "init", type: 0, maxSingleSize: 0 });
          this.worker!.postMessage({ cmd: "startRecord" });
          this.connectWebSocket();
          break;

        case "frameIndex":
          // Flow control: DVR's decoder posts frameIndex when onlyForRecord=true.
          // Send refreshFrameIndex every 8 frames to keep DVR sending data.
          this.frameIndexCount++;
          if (this.frameIndexCount <= 3 || this.frameIndexCount % 50 === 0) {
            console.log(
              `[DvrExporter] frameIndex #${this.frameIndexCount}: idx=${msg.frameIndex}, time=${msg.frameTime}`,
            );
          }
          if (msg.frameTime) {
            this.options.onProgress?.(msg.frameTime);
          }
          if (msg.frameIndex % 8 === 0 && this.canRefreshIndex) {
            this.refreshFrameIndex(msg.frameIndex);
          }
          break;

        case "getRecData": {
          const data = msg.data;
          const len = data?.length ?? 0;
          console.log(
            "[DvrExporter] getRecData:",
            len,
            "bytes, finished:",
            msg.finished,
            "manul:",
            msg.manul,
          );
          if (len > 0) {
            this.chunks.push(
              data instanceof Uint8Array ? data : new Uint8Array(data),
            );
          }
          if (msg.finished) {
            this.finishExport();
          }
          break;
        }

        case "bufferError":
          this.bufferErrorCount++;
          if (this.bufferErrorCount <= 5) {
            console.warn(
              `[DvrExporter] bufferError #${this.bufferErrorCount} (binary msg #${this.binaryMsgCount})`,
            );
          }
          break;

        case "errorCode":
          console.error(
            "[DvrExporter] decoder errorCode:",
            msg.code,
            "url:",
            msg.url,
          );
          break;
      }
    };

    this.worker.onerror = (ev) => {
      this.options.onError?.(`Export worker error: ${ev.message}`);
    };
  }

  cancel(): void {
    this.destroy();
  }

  destroy(): void {
    if (this.destroyed) return;
    this.destroyed = true;

    const elapsed = ((Date.now() - this.startedAt) / 1000).toFixed(1);
    console.log(
      `[DvrExporter] destroy — elapsed: ${elapsed}s, binary msgs: ${this.binaryMsgCount}, ` +
        `raw bytes: ${(this.totalRawBytes / 1024 / 1024).toFixed(1)}MB, ` +
        `frameIndex count: ${this.frameIndexCount}, bufferErrors: ${this.bufferErrorCount}`,
    );

    if (this.taskId && this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(cmdPlaybackClose(this.taskId)));
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    if (this.worker) {
      this.worker.postMessage({ cmd: "destroy" });
      this.worker.terminate();
      this.worker = null;
    }
    this.chunks = [];
  }

  private connectWebSocket(): void {
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${proto}//${window.location.host}/api/dvr/ws?camera_id=${this.options.cameraId}`;
    this.ws = new WebSocket(wsUrl);
    this.ws.binaryType = "arraybuffer";

    this.ws.onmessage = (ev) => {
      if (this.destroyed) return;
      if (typeof ev.data === "string") {
        this.handleTextMessage(ev.data);
      } else {
        this.handleBinaryMessage(ev.data as ArrayBuffer);
      }
    };

    this.ws.onopen = () => {
      console.log("[DvrExporter] WebSocket connected");
    };

    this.ws.onerror = () => {
      this.options.onError?.("Export WebSocket error");
    };

    this.ws.onclose = (ev) => {
      console.log(
        `[DvrExporter] WebSocket closed: code=${ev.code}, reason=${ev.reason}`,
      );
    };
  }

  private handleTextMessage(data: string): void {
    try {
      const msg = JSON.parse(data);
      const code = msg.data?.code;

      console.log("[DvrExporter] text msg:", msg.url, "code:", code);

      if (msg.url === "/device/create_connection#response" && code === 0) {
        const params: PlaybackOpenParams = {
          channelId: this.options.channelId,
          startTime: this.options.startTime,
          endTime: this.options.endTime,
          streamIndex: 0,
          backup: true,
          audio: true,
        };
        const cmd = cmdPlaybackOpen(params);
        this.taskId = cmd.data.task_id as string;
        this.canRefreshIndex = true;
        console.log("[DvrExporter] playback open (backup), task:", this.taskId);
        this.ws!.send(JSON.stringify(cmd));
      } else if (code && code !== 0 && !this.endHandled) {
        // Guard: DVR sends end code twice (video + audio). Only process once.
        this.endHandled = true;
        this.canRefreshIndex = false;
        console.log(
          `[DvrExporter] end of playback, code: ${code}, ` +
            `after ${this.binaryMsgCount} binary msgs, ${this.frameIndexCount} frameIndexes`,
        );
        this.worker?.postMessage({ cmd: "stopRecord", manul: true });
      }
    } catch {
      // non-JSON text
    }
  }

  private handleBinaryMessage(data: ArrayBuffer): void {
    if (!this.worker) return;

    this.binaryMsgCount++;
    this.totalRawBytes += data.byteLength;

    if (this.binaryMsgCount <= 3 || this.binaryMsgCount % 100 === 0) {
      console.log(
        `[DvrExporter] binary #${this.binaryMsgCount}: ${data.byteLength} bytes, total raw: ${(this.totalRawBytes / 1024 / 1024).toFixed(2)}MB`,
      );
    }

    // Feed data with onlyForRecord=true — raw compressed data goes directly
    // to AVI without decoding. Worker posts frameIndex for flow control.
    this.worker.postMessage({
      cmd: "sendData",
      buffer: data,
      onlyForRecord: true,
    });
  }

  private refreshFrameIndex(index: number): void {
    if (this.ws?.readyState === WebSocket.OPEN && this.taskId) {
      this.ws.send(JSON.stringify(cmdRefreshFrameIndex(this.taskId, index)));
    }
  }

  private async finishExport(): Promise<void> {
    const totalSize = this.chunks.reduce((sum, c) => sum + c.length, 0);
    const elapsed = ((Date.now() - this.startedAt) / 1000).toFixed(1);

    console.log(
      `[DvrExporter] finishExport — ${this.chunks.length} chunks, ` +
        `${(totalSize / 1024 / 1024).toFixed(2)}MB AVI, ` +
        `${(this.totalRawBytes / 1024 / 1024).toFixed(2)}MB raw, ` +
        `${elapsed}s elapsed, ${this.frameIndexCount} frames, ` +
        `${this.bufferErrorCount} bufferErrors`,
    );

    if (totalSize === 0) {
      this.options.onError?.("No recording data in selected range");
      this.destroy();
      return;
    }

    const aviBlob = new Blob(this.chunks as BlobPart[], { type: "video/avi" });
    const pad = (n: number) => String(n).padStart(2, "0");
    const s = new Date(this.options.startTime * 1000);
    const e = new Date(this.options.endTime * 1000);
    const baseName =
      `export_${pad(s.getHours())}-${pad(s.getMinutes())}-${pad(s.getSeconds())}` +
      `_to_${pad(e.getHours())}-${pad(e.getMinutes())}-${pad(e.getSeconds())}`;

    // Remux AVI → MP4 via server-side ffmpeg
    try {
      console.log("[DvrExporter] remuxing to MP4...");
      const resp = await fetch("/api/export/remux", {
        method: "POST",
        body: aviBlob,
      });
      if (resp.ok) {
        const mp4Blob = await resp.blob();
        console.log(
          `[DvrExporter] MP4 remux complete: ${(mp4Blob.size / 1024 / 1024).toFixed(2)}MB`,
        );
        this.options.onComplete?.(mp4Blob, `${baseName}.mp4`);
        this.destroy();
        return;
      }
      console.warn("[DvrExporter] remux failed, falling back to AVI:", resp.statusText);
    } catch (err) {
      console.warn("[DvrExporter] remux error, falling back to AVI:", err);
    }

    // Fallback: download as AVI if remux fails
    this.options.onComplete?.(aviBlob, `${baseName}.avi`);
    this.destroy();
  }
}

// Core DVR player — manages WebSocket connection, WASM decoder, and frame display.
// Ported from dvr-js/wasm-player.js

import { WebGLRenderer } from "./webgl-renderer";
import {
  cmdPlaybackOpen,
  cmdPlaybackClose,
  cmdPlaybackSeek,
  cmdKeyFrame,
  cmdAllFrame,
  cmdRefreshFrameIndex,
  msToSeconds,
  type PlaybackOpenParams,
} from "./dvr-protocol";

interface VideoFrame {
  buffer: ArrayBuffer;
  yuvLen: number;
  width: number;
  height: number;
  realTimestamp: number;
  frameIndex: number;
  frameType: number;
  notDecodeNumber: number;
}

export interface DvrPlayerOptions {
  canvas: HTMLCanvasElement;
  cameraId: string;
  onTime?: (timestamp: number) => void;
  onReady?: () => void;
  onError?: (msg: string) => void;
  onEnd?: () => void;
}

type PlayState = "STOP" | "PLAYING" | "PAUSE";

export class DvrPlayer {
  private canvas: HTMLCanvasElement;
  private cameraId: string;
  private onTime?: (timestamp: number) => void;
  private onReady?: () => void;
  private onError?: (msg: string) => void;
  private onEnd?: () => void;

  private ws: WebSocket | null = null;
  private renderer: WebGLRenderer | null = null;
  private decodeWorker: Worker | null = null;

  private taskId = "";
  private playState: PlayState = "STOP";
  private playSpeed = 1;
  private isKeyFramePlay = false;

  private videoQueue: VideoFrame[] = [];
  private maxVideoQueueLength = 8;
  private isDecoding = false;
  private isFirstDecode = false;

  private basicRealTime = 0;
  private basicFrameTime = 0;
  private frameTimestamp = 0;
  private displayLoopId: number | null = null;
  private isEndPlay = false;
  private notDecodeNumber = 0;

  constructor(options: DvrPlayerOptions) {
    this.canvas = options.canvas;
    this.cameraId = options.cameraId;
    this.onTime = options.onTime;
    this.onReady = options.onReady;
    this.onError = options.onError;
    this.onEnd = options.onEnd;
  }

  play(params: PlaybackOpenParams): void {
    this.isEndPlay = false;
    this.notDecodeNumber = 0;
    this.isFirstDecode = false;

    // Init renderer
    if (!this.renderer) {
      this.renderer = new WebGLRenderer(this.canvas);
    }

    // Init decoder worker
    this.initDecoder();

    // Connect WebSocket
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${proto}//${window.location.host}/api/dvr/ws?camera_id=${this.cameraId}`;
    this.ws = new WebSocket(wsUrl);
    this.ws.binaryType = "arraybuffer";

    this.ws.onopen = () => {
      console.log("[DvrPlayer] WebSocket connected");
    };

    this.ws.onmessage = (ev) => {
      if (typeof ev.data === "string") {
        this.handleTextMessage(ev.data, params);
      } else {
        this.handleBinaryMessage(ev.data as ArrayBuffer);
      }
    };

    this.ws.onerror = () => {
      this.onError?.("WebSocket connection error");
    };

    this.ws.onclose = () => {
      // Connection closed — player handles cleanup via destroy()
    };
  }

  stop(): void {
    if (this.taskId && this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(cmdPlaybackClose(this.taskId)));
    }
    this.stopInternal();
  }

  seek(timestampMs: number): void {
    if (!this.taskId || !this.ws || this.ws.readyState !== WebSocket.OPEN)
      return;

    this.stopDecode();
    this.clearFrameList();
    this.renderer?.clear();

    const cmd = cmdPlaybackSeek(
      this.taskId,
      msToSeconds(timestampMs),
    );
    this.ws.send(JSON.stringify(cmd));

    this.isFirstDecode = false;
    this.resetBasicTime();

    // Re-apply speed mode after seek
    if (this.playSpeed > 2) {
      const kfCmd = cmdKeyFrame(
        this.taskId,
        msToSeconds(timestampMs),
      );
      this.ws.send(JSON.stringify(kfCmd));
      this.isKeyFramePlay = true;
    }

    this.startDecode();
    this.startDisplayLoop();
  }

  setSpeed(speed: number): void {
    if (!this.taskId || !this.ws || this.ws.readyState !== WebSocket.OPEN) {
      this.playSpeed = speed;
      return;
    }

    const frameTimeSec = msToSeconds(this.frameTimestamp || Date.now());

    if (speed > 2) {
      if (this.playSpeed <= 2 || !this.isKeyFramePlay) {
        this.stopDecode();
        this.clearFrameList();
        this.renderer?.clear();
        this.isKeyFramePlay = true;
        this.ws.send(JSON.stringify(cmdKeyFrame(this.taskId, frameTimeSec)));
        this.startDecode();
      }
    } else {
      if (this.playSpeed > 2 || this.isKeyFramePlay) {
        this.stopDecode();
        this.clearFrameList();
        this.renderer?.clear();
        this.isKeyFramePlay = false;
        this.ws.send(JSON.stringify(cmdAllFrame(this.taskId, frameTimeSec)));
        this.startDecode();
      }
    }

    this.playSpeed = speed;
    this.resetBasicTime();
  }

  destroy(): void {
    this.stop();
    this.renderer?.destroy();
    this.renderer = null;
    if (this.decodeWorker) {
      this.decodeWorker.postMessage({ cmd: "destroy" });
      this.decodeWorker.terminate();
      this.decodeWorker = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  // --- Internal ---

  private handleTextMessage(data: string, params: PlaybackOpenParams): void {
    try {
      const msg = JSON.parse(data);
      const code = msg.data?.code;
      console.log("[DvrPlayer] text message:", msg.url, "code:", code);

      if (msg.url === "/device/create_connection#response" && code === 0) {
        // DVR authenticated — send playback open
        const cmd = cmdPlaybackOpen(params);
        this.taskId = cmd.data.task_id as string;
        this.playState = "PLAYING";
        console.log("[DvrPlayer] sending playback open:", JSON.stringify(cmd));
        this.ws!.send(JSON.stringify(cmd));
        this.onReady?.();
      } else if (
        msg.url === "/device/playback/data" &&
        code === 0x20000011
      ) {
        // End of playback data
        this.handleEndPlay();
      }
    } catch {
      console.log("[DvrPlayer] non-JSON text message:", data.substring(0, 200));
    }
  }

  private binaryCount = 0;
  private handleBinaryMessage(data: ArrayBuffer): void {
    if (!this.decodeWorker) return;
    this.binaryCount++;
    if (this.binaryCount <= 3 || this.binaryCount % 100 === 0) {
      console.log(`[DvrPlayer] binary message #${this.binaryCount}, size=${data.byteLength}`);
    }

    // Feed binary data to decoder
    this.decodeWorker.postMessage(
      { cmd: "sendData", buffer: data },
    );

    if (!this.isFirstDecode) {
      this.startDecode();
      this.isFirstDecode = true;
      this.startDisplayLoop();
    }

    if (!this.isDecoding) {
      this.startDecode();
    }
  }

  private initDecoder(): void {
    if (this.decodeWorker) {
      this.decodeWorker.terminate();
    }

    this.decodeWorker = new Worker("/dvr-decoder/decoder.js");

    this.decodeWorker.onmessage = (ev) => {
      const msg = ev.data;
      if (!msg?.cmd) return;

      console.log("[DvrPlayer] decoder msg:", msg.cmd, msg.cmd === "getVideoFrame" ? `${msg.data?.width}x${msg.data?.height}` : "");

      switch (msg.cmd) {
        case "ready":
          console.log("[DvrPlayer] decoder ready, sending init");
          this.decodeWorker!.postMessage({ cmd: "init", type: 0 });
          break;

        case "getVideoFrame":
          this.onVideoFrame(msg.data);
          break;

        case "bufferError":
          console.warn("[DvrPlayer] decoder buffer error");
          this.stopDecode();
          break;

        case "errorCode":
          this.onError?.(`Decoder error: code=${msg.code}`);
          break;
      }
    };

    this.decodeWorker.onerror = (ev) => {
      console.error("[DvrPlayer] decoder worker error:", ev.message, ev);
    };
  }

  private onVideoFrame(frame: VideoFrame): void {
    this.notDecodeNumber = frame.notDecodeNumber;

    // Frame index flow control — refresh every 8 frames or on keyframe play
    if (frame.frameIndex > 0) {
      if (frame.frameIndex % 8 === 0 || this.isKeyFramePlay) {
        this.refreshFrameIndex(frame.frameIndex);
      }
    }

    // Skip non-visible frames (frameType 4 = audio-only or metadata)
    if (frame.frameType === 4) return;

    this.videoQueue.push(frame);
    if (this.videoQueue.length > this.maxVideoQueueLength) {
      this.stopDecode();
    }
  }

  private displayLoop = (): void => {
    if (this.playState !== "PLAYING") return;
    this.displayLoopId = requestAnimationFrame(this.displayLoop);

    if (this.videoQueue.length === 0) return;

    // Try to display up to 2 frames per loop iteration
    for (let i = 0; i < 2; i++) {
      if (this.videoQueue.length === 0) break;
      if (this.displayNextFrame()) {
        this.videoQueue.shift();
      } else {
        break;
      }
    }

    // Resume decoding if queue is draining
    if (
      this.videoQueue.length < this.maxVideoQueueLength / 2 &&
      !this.isDecoding
    ) {
      this.startDecode();
    }
  };

  private displayNextFrame(): boolean {
    const frame = this.videoQueue[0];
    const now = Date.now();
    const frameTime = frame.realTimestamp;

    if (!this.basicFrameTime) {
      this.basicRealTime = now;
      this.basicFrameTime = frameTime;
    }

    const realElapsed = now - this.basicRealTime;
    const frameElapsed = frameTime - this.basicFrameTime;

    // Frame time went backwards — reset baseline
    if (frameElapsed < 0) {
      this.basicFrameTime = frameTime;
    }

    // Speed-aware timing: don't display frame until enough real time has passed
    if (realElapsed * this.playSpeed < frameElapsed) {
      this.onTime?.(frame.realTimestamp);
      return false;
    }

    // Render the frame
    this.renderFrame(frame);

    // Reset baseline if too far out of sync
    if (realElapsed > 10000 * this.playSpeed) {
      this.resetBasicTime();
    }

    this.onTime?.(frame.realTimestamp);
    return true;
  }

  private renderFrame(frame: VideoFrame): void {
    if (!this.renderer) return;
    const yuvData = new Uint8Array(frame.buffer);
    const yLen = frame.width * frame.height;
    const uvLen = (frame.width / 2) * (frame.height / 2);
    this.renderer.renderFrame(yuvData, frame.width, frame.height, yLen, uvLen);
    this.frameTimestamp = frame.realTimestamp;
  }

  private refreshFrameIndex(index: number): void {
    if (
      (this.playState === "PLAYING") &&
      this.ws?.readyState === WebSocket.OPEN
    ) {
      const cmd = cmdRefreshFrameIndex(this.taskId, index);
      this.ws.send(JSON.stringify(cmd));
    }
  }

  private startDecode(): void {
    this.isDecoding = true;
    this.decodeWorker?.postMessage({ cmd: "decodeFrame" });
  }

  private stopDecode(): void {
    this.isDecoding = false;
    this.decodeWorker?.postMessage({ cmd: "stopDecode" });
  }

  private clearFrameList(): void {
    this.videoQueue = [];
    this.decodeWorker?.postMessage({ cmd: "clear" });
  }

  private startDisplayLoop(): void {
    this.clearDisplayLoop();
    this.displayLoopId = requestAnimationFrame(this.displayLoop);
  }

  private clearDisplayLoop(): void {
    if (this.displayLoopId !== null) {
      cancelAnimationFrame(this.displayLoopId);
      this.displayLoopId = null;
    }
  }

  private resetBasicTime(): void {
    this.basicRealTime = 0;
    this.basicFrameTime = 0;
  }

  private stopInternal(): void {
    this.playState = "STOP";
    this.stopDecode();
    this.clearFrameList();
    this.clearDisplayLoop();
    this.renderer?.clear();
    this.taskId = "";
  }

  private handleEndPlay(): void {
    if (this.isEndPlay) return;
    this.isEndPlay = true;

    if (this.notDecodeNumber === 0) {
      this.playState = "PAUSE";
      this.stopDecode();
      this.onEnd?.();
    } else {
      // Wait for remaining frames to decode
      setTimeout(() => {
        this.isEndPlay = false;
        this.handleEndPlay();
      }, 50);
    }
  }
}

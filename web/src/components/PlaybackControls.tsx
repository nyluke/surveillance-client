import { useState, useRef, useEffect } from "react";
import { useCameraStore } from "../stores/cameraStore";
import { DvrExporter } from "../lib/dvr-exporter";
import { msToSeconds } from "../lib/dvr-protocol";
import type { DvrPlayerHandle } from "./DvrPlayer";

const DURATIONS = [
  { value: 900, label: "15m" },
  { value: 1800, label: "30m" },
  { value: 3600, label: "1h" },
  { value: 7200, label: "2h" },
  { value: 14400, label: "4h" },
];

const SPEEDS = [1, 2, 4, 8, 16, 32];

const SKIPS = [
  { seconds: -300, label: "-5m" },
  { seconds: -60, label: "-1m" },
  { seconds: -10, label: "-10s" },
  { seconds: 10, label: "+10s" },
  { seconds: 60, label: "+1m" },
  { seconds: 300, label: "+5m" },
];

function formatTime(ts: number): string {
  const d = new Date(ts);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function formatTimeForFilename(ts: number): string {
  const d = new Date(ts);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getHours())}-${pad(d.getMinutes())}-${pad(d.getSeconds())}`;
}

export default function PlaybackControls({
  cameraName,
  cameraId,
  channelId,
  playerRef,
  currentTime,
}: {
  cameraName: string;
  cameraId: string;
  channelId: string;
  playerRef: React.RefObject<DvrPlayerHandle | null>;
  currentTime: number | null;
}) {
  const {
    playbackDate,
    playbackTime,
    playbackDuration,
    playbackSpeed,
    playbackActive,
    setPlaybackDate,
    setPlaybackTime,
    setPlaybackDuration,
    setPlaybackSpeed,
    setPlaybackActive,
    stopPlayback,
  } = useCameraStore();

  const [loading, setLoading] = useState(false);
  const [startMark, setStartMark] = useState<number | null>(null);
  const [endMark, setEndMark] = useState<number | null>(null);
  const [exporting, setExporting] = useState(false);
  const [exportProgress, setExportProgress] = useState<number | null>(null);
  const exporterRef = useRef<DvrExporter | null>(null);

  const canPlay = playbackDate && playbackTime && !loading;

  const handlePlay = () => {
    setLoading(true);
    setPlaybackActive(true);
  };

  const handleStop = () => {
    playerRef.current?.stop();
    stopPlayback();
    setLoading(false);
  };

  const handleSpeedChange = (speed: number) => {
    setPlaybackSpeed(speed);
    playerRef.current?.setSpeed(speed);
  };

  const handleSkip = (seconds: number) => {
    if (currentTime) {
      playerRef.current?.seek(currentTime + seconds * 1000);
    }
  };

  const handleCapture = () => {
    const ts = currentTime
      ? formatTimeForFilename(currentTime)
      : String(Date.now());
    playerRef.current?.captureFrame(`${cameraName}_${ts}.jpg`);
  };

  const handleMarkIn = () => {
    if (startMark !== null) {
      setStartMark(null);
    } else if (currentTime) {
      setStartMark(currentTime);
      if (endMark !== null && endMark <= currentTime) {
        setEndMark(null);
      }
    }
  };

  const handleMarkOut = () => {
    if (endMark !== null) {
      setEndMark(null);
    } else if (currentTime) {
      setEndMark(currentTime);
      if (startMark !== null && startMark >= currentTime) {
        setStartMark(null);
      }
    }
  };

  const canExport = startMark !== null && endMark !== null && !exporting;

  const handleExport = () => {
    if (!canExport) return;

    const exportStartMs = Math.min(startMark, endMark);
    const exportEndMs = Math.max(startMark, endMark);

    const exporter = new DvrExporter({
      cameraId,
      channelId,
      startTime: msToSeconds(exportStartMs),
      endTime: msToSeconds(exportEndMs),
      onProgress: (frameTimeMs) => {
        setExportProgress(frameTimeMs);
      },
      onComplete: (blob, filename) => {
        // Trigger download
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${cameraName}_${filename}`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        setExporting(false);
        setExportProgress(null);
        exporterRef.current = null;
      },
      onError: (msg) => {
        console.error("[Export]", msg);
        setExporting(false);
        setExportProgress(null);
        exporterRef.current = null;
      },
    });

    exporter.start();
    exporterRef.current = exporter;
    setExporting(true);
    setExportProgress(null);
  };

  const handleCancelExport = () => {
    exporterRef.current?.cancel();
    exporterRef.current = null;
    setExporting(false);
    setExportProgress(null);
  };

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      exporterRef.current?.cancel();
      exporterRef.current = null;
    };
  }, []);

  // Clear loading when playback becomes active
  if (playbackActive && loading) {
    setLoading(false);
  }

  return (
    <div className="px-4 py-2 bg-gray-900 border-t border-gray-800 flex items-center gap-3 text-sm">
      <span className="text-gray-300 font-medium truncate shrink-0">
        {cameraName}
      </span>
      <div className="w-px h-5 bg-gray-700" />
      <input
        type="date"
        value={playbackDate ?? ""}
        onChange={(e) => setPlaybackDate(e.target.value)}
        className="bg-gray-800 border border-gray-700 rounded px-2 py-1 text-gray-200 text-sm focus:outline-none focus:border-blue-500"
      />
      <input
        type="time"
        value={playbackTime ?? ""}
        onChange={(e) => setPlaybackTime(e.target.value)}
        className="bg-gray-800 border border-gray-700 rounded px-2 py-1 text-gray-200 text-sm focus:outline-none focus:border-blue-500"
      />
      <select
        value={playbackDuration}
        onChange={(e) => setPlaybackDuration(Number(e.target.value))}
        className="bg-gray-800 border border-gray-700 rounded px-2 py-1 text-gray-200 text-sm focus:outline-none focus:border-blue-500"
      >
        {DURATIONS.map((d) => (
          <option key={d.value} value={d.value}>
            {d.label}
          </option>
        ))}
      </select>
      {playbackActive ? (
        <>
          {currentTime && (
            <span className="text-gray-400 font-mono text-xs tabular-nums">
              {formatTime(currentTime)}
            </span>
          )}
          <div className="flex gap-0.5 bg-gray-800 rounded p-0.5">
            {SKIPS.map((s) => (
              <button
                key={s.seconds}
                onClick={() => handleSkip(s.seconds)}
                className="px-1.5 py-0.5 rounded text-xs font-medium text-gray-400 hover:text-gray-200 hover:bg-gray-700 transition-colors"
              >
                {s.label}
              </button>
            ))}
          </div>
          <div className="flex gap-0.5 bg-gray-800 rounded p-0.5">
            {SPEEDS.map((s) => (
              <button
                key={s}
                onClick={() => handleSpeedChange(s)}
                className={`px-1.5 py-0.5 rounded text-xs font-medium transition-colors ${
                  playbackSpeed === s
                    ? "bg-blue-600 text-white"
                    : "text-gray-400 hover:text-gray-200"
                }`}
              >
                {s}x
              </button>
            ))}
          </div>
          <div className="w-px h-5 bg-gray-700" />
          <button
            onClick={handleCapture}
            className="px-2 py-1 rounded bg-gray-700 hover:bg-gray-600 text-gray-200 text-sm font-medium transition-colors shrink-0"
            title="Capture current frame as JPEG"
          >
            Capture
          </button>
          <div className="flex gap-0.5 bg-gray-800 rounded p-0.5">
            <button
              onClick={handleMarkIn}
              className={`px-2 py-0.5 rounded text-xs font-medium transition-colors ${
                startMark !== null
                  ? "bg-green-700 text-green-100"
                  : "text-gray-400 hover:text-gray-200 hover:bg-gray-700"
              }`}
              title={
                startMark !== null
                  ? `In: ${formatTime(startMark)} (click to clear)`
                  : "Mark clip start"
              }
            >
              In{startMark !== null ? ` ${formatTime(startMark)}` : ""}
            </button>
            <button
              onClick={handleMarkOut}
              className={`px-2 py-0.5 rounded text-xs font-medium transition-colors ${
                endMark !== null
                  ? "bg-green-700 text-green-100"
                  : "text-gray-400 hover:text-gray-200 hover:bg-gray-700"
              }`}
              title={
                endMark !== null
                  ? `Out: ${formatTime(endMark)} (click to clear)`
                  : "Mark clip end"
              }
            >
              Out{endMark !== null ? ` ${formatTime(endMark)}` : ""}
            </button>
          </div>
          {exporting ? (
            <>
              <span className="text-yellow-400 font-medium animate-pulse">
                Exporting...
              </span>
              {exportProgress && (
                <span className="text-gray-400 font-mono text-xs tabular-nums">
                  {formatTime(exportProgress)}
                </span>
              )}
              <button
                onClick={handleCancelExport}
                className="px-2 py-1 rounded bg-red-600 hover:bg-red-500 text-white text-sm font-medium transition-colors shrink-0"
              >
                Cancel
              </button>
            </>
          ) : (
            <button
              onClick={handleExport}
              disabled={!canExport}
              className="px-2 py-1 rounded bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-medium transition-colors shrink-0"
              title="Export clip between In and Out marks as AVI"
            >
              Export
            </button>
          )}
          <button
            onClick={handleStop}
            className="px-3 py-1 rounded bg-red-600 hover:bg-red-500 text-white text-sm font-medium transition-colors shrink-0"
          >
            Stop
          </button>
        </>
      ) : (
        <button
          onClick={handlePlay}
          disabled={!canPlay}
          className="px-3 py-1 rounded bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-medium transition-colors"
        >
          {loading ? "Loading..." : "Play"}
        </button>
      )}
    </div>
  );
}

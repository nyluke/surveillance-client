import { useState } from "react";
import { useCameraStore } from "../stores/cameraStore";
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

export default function PlaybackControls({
  cameraName,
  playerRef,
  currentTime,
}: {
  cameraName: string;
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

  const canPlay = playbackDate && playbackTime && !loading;

  const handlePlay = () => {
    setLoading(true);
    setPlaybackActive(true);
    // Loading state will be cleared by onReady from DvrPlayer
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

  // Clear loading when playback becomes active (DvrPlayer calls onReady)
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

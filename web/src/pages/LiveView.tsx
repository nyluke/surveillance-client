import { useEffect, useRef, useState, useCallback } from "react";
import { useCameraStore } from "../stores/cameraStore";
import CameraSidebar from "../components/CameraSidebar";
import VideoPlayer from "../components/VideoPlayer";
import QuadView from "../components/QuadView";
import PlaybackControls from "../components/PlaybackControls";
import DvrPlayer, { type DvrPlayerHandle } from "../components/DvrPlayer";
import { channelToGuid } from "../lib/dvr-protocol";

function extractChannelId(rtspUrl: string): number {
  const match = rtspUrl.match(/chID=(\d+)/);
  return match ? parseInt(match[1], 10) : 1;
}

export default function LiveView() {
  const {
    cameras,
    loading,
    error,
    viewMode,
    selectedCamera,
    appMode,
    playbackActive,
    playbackDate,
    playbackTime,
    playbackDuration,
    fetchCameras,
    fetchGroups,
    setPlaybackActive,
  } = useCameraStore();

  const dvrPlayerRef = useRef<DvrPlayerHandle | null>(null);
  const [currentTime, setCurrentTime] = useState<number | null>(null);

  useEffect(() => {
    fetchCameras();
    fetchGroups();
  }, [fetchCameras, fetchGroups]);

  const enabledCameras = cameras.filter((c) => c.enabled);

  // Auto-select first camera if none selected
  useEffect(() => {
    if (enabledCameras.length > 0 && !selectedCamera) {
      useCameraStore.getState().selectCamera(enabledCameras[0].id);
    }
  }, [enabledCameras, selectedCamera]);

  const handleTimeUpdate = useCallback((ts: number) => {
    setCurrentTime(ts);
  }, []);

  const handleReady = useCallback(() => {
    setPlaybackActive(true);
  }, [setPlaybackActive]);

  const handleError = useCallback((msg: string) => {
    console.error("DVR playback error:", msg);
  }, []);

  if (loading && cameras.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-gray-400">Loading cameras...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-red-400">Error: {error}</p>
      </div>
    );
  }

  if (enabledCameras.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-gray-400">
          No cameras configured. Go to Settings to add cameras.
        </p>
      </div>
    );
  }

  const activeCam = cameras.find((c) => c.id === selectedCamera);
  const isPlayback = appMode === "playback";

  // Compute playback params for DvrPlayer
  let playbackStartTime = 0;
  let playbackEndTime = 0;
  let channelId = channelToGuid(1);
  if (isPlayback && activeCam && playbackDate && playbackTime) {
    const start = new Date(`${playbackDate}T${playbackTime}:00`);
    const end = new Date(start.getTime() + playbackDuration * 1000);
    playbackStartTime = Math.floor(start.getTime() / 1000);
    playbackEndTime = Math.floor(end.getTime() / 1000);
    channelId = channelToGuid(extractChannelId(activeCam.rtsp_main));
  }

  return (
    <div className="flex h-[calc(100vh-57px)]">
      <CameraSidebar cameras={enabledCameras} />
      <div className="flex-1 flex flex-col min-w-0">
        <div className="flex-1 relative bg-black">
          {isPlayback ? (
            selectedCamera && playbackActive && playbackStartTime ? (
              <DvrPlayer
                key={`dvr-${selectedCamera}-${playbackStartTime}`}
                ref={dvrPlayerRef}
                cameraId={selectedCamera}
                channelId={channelId}
                startTime={playbackStartTime}
                endTime={playbackEndTime}
                onTimeUpdate={handleTimeUpdate}
                onReady={handleReady}
                onError={handleError}
                className="absolute inset-0"
              />
            ) : (
              <div className="absolute inset-0 flex items-center justify-center">
                <p className="text-gray-600">
                  {selectedCamera
                    ? "Select date/time and press Play"
                    : "Select a camera"}
                </p>
              </div>
            )
          ) : viewMode === "single" && selectedCamera ? (
            <VideoPlayer
              key={selectedCamera}
              cameraId={selectedCamera}
              className="absolute inset-0"
            />
          ) : viewMode === "quad" ? (
            <QuadView />
          ) : (
            <div className="absolute inset-0 flex items-center justify-center">
              <p className="text-gray-600">Select a camera</p>
            </div>
          )}
        </div>
        {isPlayback && activeCam ? (
          <PlaybackControls
            cameraName={activeCam.name}
            cameraId={activeCam.id}
            channelId={channelId}
            playerRef={dvrPlayerRef}
            currentTime={currentTime}
          />
        ) : !isPlayback && viewMode === "single" && activeCam ? (
          <div className="px-4 py-2 bg-gray-900 border-t border-gray-800 text-sm text-gray-300">
            {activeCam.name}
          </div>
        ) : null}
      </div>
    </div>
  );
}

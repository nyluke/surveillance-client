import { useCameraStore } from "../stores/cameraStore";
import VideoPlayer from "./VideoPlayer";

export default function QuadView() {
  const { quadCameras, cameras, clearQuadSlot } = useCameraStore();

  function handleCellClick(cameraId: string) {
    useCameraStore.setState({ selectedCamera: cameraId, viewMode: "single" });
  }

  return (
    <div className="absolute inset-0 grid grid-cols-2 grid-rows-2 gap-1 p-1">
      {quadCameras.map((camId, idx) => {
        if (!camId) {
          return (
            <div
              key={idx}
              className="bg-gray-900 rounded-lg flex items-center justify-center"
            >
              <span className="text-gray-600 text-sm">
                Click a camera to assign
              </span>
            </div>
          );
        }

        const cam = cameras.find((c) => c.id === camId);
        if (!cam) return null;

        return (
          <div
            key={camId}
            className="relative bg-black rounded-lg overflow-hidden cursor-pointer group"
            onClick={() => handleCellClick(camId)}
          >
            <VideoPlayer cameraId={camId} useSub />
            <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent p-2 pt-6">
              <span className="text-xs font-medium text-white truncate block">
                {cam.name}
              </span>
            </div>
            <button
              onClick={(e) => {
                e.stopPropagation();
                clearQuadSlot(idx);
              }}
              className="absolute top-1 right-1 w-5 h-5 rounded bg-black/60 text-gray-400 hover:text-white text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
            >
              ×
            </button>
          </div>
        );
      })}
    </div>
  );
}

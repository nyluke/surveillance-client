import type { Camera } from "../lib/types";
import { useCameraStore } from "../stores/cameraStore";

const QUAD_LABELS = ["\u2460", "\u2461", "\u2462", "\u2463"];

export default function CameraSidebar({ cameras }: { cameras: Camera[] }) {
  const { viewMode, selectedCamera, quadCameras, selectCamera } =
    useCameraStore();

  return (
    <div className="w-48 shrink-0 bg-gray-900 border-r border-gray-800 overflow-y-auto">
      {cameras.map((cam) => {
        const isSelected = viewMode === "single" && selectedCamera === cam.id;
        const quadIdx = quadCameras.indexOf(cam.id);
        const isInQuad = viewMode === "quad" && quadIdx !== -1;

        return (
          <button
            key={cam.id}
            onClick={() => selectCamera(cam.id)}
            className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2 transition-colors ${
              isSelected
                ? "bg-blue-600/20 text-blue-400"
                : isInQuad
                  ? "bg-blue-600/10 text-blue-300"
                  : "text-gray-400 hover:bg-gray-800 hover:text-gray-200"
            }`}
          >
            <span className="w-5 text-center shrink-0">
              {isSelected && "\u25CF"}
              {isInQuad && QUAD_LABELS[quadIdx]}
            </span>
            <span className="truncate">{cam.name}</span>
          </button>
        );
      })}
    </div>
  );
}

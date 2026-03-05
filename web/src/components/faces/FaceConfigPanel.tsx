import { useEffect, useState } from "react";
import { useFaceStore } from "../../stores/faceStore";
import { useCameraStore } from "../../stores/cameraStore";
import type { FaceMonitorConfig } from "../../lib/types";

export default function FaceConfigPanel() {
  const { configs, fetchConfig, saveConfig, status, fetchStatus } =
    useFaceStore();
  const { cameras, fetchCameras } = useCameraStore();
  const [localConfigs, setLocalConfigs] = useState<
    Record<string, FaceMonitorConfig>
  >({});
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    fetchConfig();
    fetchStatus();
    fetchCameras();
  }, [fetchConfig, fetchStatus, fetchCameras]);

  useEffect(() => {
    const map: Record<string, FaceMonitorConfig> = {};
    for (const c of configs) {
      map[c.camera_id] = c;
    }
    setLocalConfigs(map);
    setDirty(false);
  }, [configs]);

  const setMonitorType = (
    cameraId: string,
    type: "" | "realtime" | "batch" | "both",
  ) => {
    setDirty(true);
    if (type === "") {
      const next = { ...localConfigs };
      delete next[cameraId];
      setLocalConfigs(next);
    } else {
      setLocalConfigs((prev) => ({
        ...prev,
        [cameraId]: {
          camera_id: cameraId,
          monitor_type: type,
          interval_seconds: prev[cameraId]?.interval_seconds || 2,
        },
      }));
    }
  };

  const handleSave = async () => {
    await saveConfig(Object.values(localConfigs));
    setDirty(false);
  };

  return (
    <div className="space-y-6">
      {/* Service status */}
      <div>
        <h3 className="text-sm font-medium text-gray-400 uppercase tracking-wide mb-2">
          Service Status
        </h3>
        <div className="bg-gray-800 rounded-lg p-4 space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-gray-400">Face Service</span>
            <span>
              {status?.face_service_configured ? (
                status.face_service_online ? (
                  <span className="text-green-400">Online</span>
                ) : (
                  <span className="text-red-400">
                    Offline{" "}
                    {status.face_service_error && (
                      <span className="text-gray-500">
                        ({status.face_service_error})
                      </span>
                    )}
                  </span>
                )
              ) : (
                <span className="text-gray-500">Not configured</span>
              )}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-400">Slack Alerts</span>
            <span>
              {status?.slack_configured ? (
                <span className="text-green-400">Configured</span>
              ) : (
                <span className="text-gray-500">Not configured</span>
              )}
            </span>
          </div>
        </div>
      </div>

      {/* Camera monitoring */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-sm font-medium text-gray-400 uppercase tracking-wide">
            Camera Monitoring
          </h3>
          {dirty && (
            <button
              onClick={handleSave}
              className="px-3 py-1.5 bg-blue-600 rounded text-sm font-medium hover:bg-blue-500"
            >
              Save Changes
            </button>
          )}
        </div>
        <div className="space-y-1">
          {cameras.map((cam) => (
            <div
              key={cam.id}
              className="flex items-center justify-between bg-gray-800 rounded px-3 py-2"
            >
              <span className="text-sm truncate flex-1">{cam.name}</span>
              <select
                value={localConfigs[cam.id]?.monitor_type || ""}
                onChange={(e) =>
                  setMonitorType(
                    cam.id,
                    e.target.value as "" | "realtime" | "batch" | "both",
                  )
                }
                className="bg-gray-700 rounded px-2 py-1 text-xs ml-2"
              >
                <option value="">Off</option>
                <option value="realtime">Realtime (alerts)</option>
                <option value="batch">Batch (visitors)</option>
                <option value="both">Both</option>
              </select>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

import { useEffect } from "react";
import { useFaceStore } from "../../stores/faceStore";

export default function AlertsPanel() {
  const { sightings, fetchSightings } = useFaceStore();

  useEffect(() => {
    fetchSightings();
    const interval = setInterval(() => fetchSightings(), 10000);
    return () => clearInterval(interval);
  }, [fetchSightings]);

  return (
    <div>
      <h2 className="text-sm font-medium text-gray-400 uppercase tracking-wide mb-4">
        Recent Sightings ({sightings.length})
      </h2>

      {sightings.length === 0 && (
        <p className="text-gray-500 text-sm">No sightings yet.</p>
      )}

      <div className="space-y-2">
        {sightings.map((s) => (
          <div
            key={s.id}
            className="flex items-center gap-3 bg-gray-800 rounded-lg p-3"
          >
            {s.crop_url ? (
              <img
                src={s.crop_url}
                alt=""
                className="w-12 h-12 rounded object-cover flex-shrink-0"
              />
            ) : (
              <div className="w-12 h-12 rounded bg-gray-700 flex-shrink-0" />
            )}
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium">{s.subject_name}</p>
              <p className="text-xs text-gray-400">
                {s.camera_name} &middot;{" "}
                {(s.confidence * 100).toFixed(0)}% confidence
              </p>
            </div>
            <span className="text-xs text-gray-500 flex-shrink-0">
              {new Date(s.seen_at + "Z").toLocaleString()}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

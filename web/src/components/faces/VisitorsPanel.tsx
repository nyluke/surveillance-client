import { useEffect, useState } from "react";
import { useFaceStore } from "../../stores/faceStore";

export default function VisitorsPanel() {
  const { clusters, fetchClusters, labelCluster, triggerClustering, loading } =
    useFaceStore();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editLabel, setEditLabel] = useState("");

  useEffect(() => {
    fetchClusters();
  }, [fetchClusters]);

  const handleLabel = async (id: string) => {
    await labelCluster(id, editLabel);
    setEditingId(null);
    setEditLabel("");
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-medium text-gray-400 uppercase tracking-wide">
          Visitor Clusters ({clusters.length})
        </h2>
        <button
          onClick={() => triggerClustering()}
          disabled={loading}
          className="px-3 py-1.5 bg-blue-600 rounded text-sm font-medium hover:bg-blue-500 disabled:opacity-50"
        >
          {loading ? "Clustering..." : "Run Clustering"}
        </button>
      </div>

      {clusters.length === 0 && (
        <p className="text-gray-500 text-sm">
          No visitor clusters yet. Enable batch monitoring on cameras and run
          clustering.
        </p>
      )}

      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
        {clusters.map((c) => (
          <div
            key={c.id}
            className="bg-gray-800 rounded-lg overflow-hidden"
          >
            {c.representative_crop ? (
              <img
                src={c.representative_crop}
                alt={c.label || c.id}
                className="w-full aspect-square object-cover"
              />
            ) : (
              <div className="w-full aspect-square bg-gray-700 flex items-center justify-center text-gray-500 text-2xl">
                ?
              </div>
            )}
            <div className="p-2">
              {editingId === c.id ? (
                <div className="flex gap-1">
                  <input
                    value={editLabel}
                    onChange={(e) => setEditLabel(e.target.value)}
                    className="flex-1 bg-gray-700 rounded px-2 py-1 text-xs min-w-0"
                    autoFocus
                    onKeyDown={(e) => e.key === "Enter" && handleLabel(c.id)}
                  />
                  <button
                    onClick={() => handleLabel(c.id)}
                    className="text-xs text-blue-400"
                  >
                    Save
                  </button>
                </div>
              ) : (
                <p
                  className="text-sm font-medium truncate cursor-pointer hover:text-blue-400"
                  onClick={() => {
                    setEditingId(c.id);
                    setEditLabel(c.label || "");
                  }}
                >
                  {c.label || "Unlabeled"}
                </p>
              )}
              <p className="text-xs text-gray-500 mt-1">
                {c.visit_count} visit{c.visit_count !== 1 ? "s" : ""}
              </p>
              <p className="text-xs text-gray-500">
                Last: {new Date(c.last_seen + "Z").toLocaleDateString()}
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

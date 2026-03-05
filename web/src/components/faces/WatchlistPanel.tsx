import { useEffect, useState } from "react";
import { useFaceStore } from "../../stores/faceStore";
import AddSubjectDialog from "./AddSubjectDialog";

export default function WatchlistPanel() {
  const { subjects, fetchSubjects, deleteSubject, loading } = useFaceStore();
  const [showAdd, setShowAdd] = useState(false);

  useEffect(() => {
    fetchSubjects();
  }, [fetchSubjects]);

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-sm font-medium text-gray-400 uppercase tracking-wide">
          Watchlist ({subjects.length})
        </h2>
        <button
          onClick={() => setShowAdd(true)}
          className="px-3 py-1.5 bg-blue-600 rounded text-sm font-medium hover:bg-blue-500"
        >
          Add Person
        </button>
      </div>

      {loading && subjects.length === 0 && (
        <p className="text-gray-500 text-sm">Loading...</p>
      )}

      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
        {subjects.map((sub) => (
          <div
            key={sub.id}
            className="bg-gray-800 rounded-lg overflow-hidden group"
          >
            {sub.crop_url ? (
              <img
                src={sub.crop_url}
                alt={sub.name}
                className="w-full aspect-square object-cover"
              />
            ) : (
              <div className="w-full aspect-square bg-gray-700 flex items-center justify-center text-gray-500 text-2xl">
                ?
              </div>
            )}
            <div className="p-2">
              <p className="text-sm font-medium truncate">{sub.name}</p>
              {sub.notes && (
                <p className="text-xs text-gray-500 truncate">{sub.notes}</p>
              )}
              <div className="flex items-center justify-between mt-1">
                <span
                  className={`text-xs ${sub.alert_enabled ? "text-green-400" : "text-gray-500"}`}
                >
                  {sub.alert_enabled ? "Alerts on" : "Alerts off"}
                </span>
                <button
                  onClick={() => {
                    if (confirm(`Delete ${sub.name}?`)) deleteSubject(sub.id);
                  }}
                  className="text-xs text-red-400 opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  Delete
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {showAdd && <AddSubjectDialog onClose={() => setShowAdd(false)} />}
    </div>
  );
}

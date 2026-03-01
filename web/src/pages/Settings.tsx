import { useEffect, useState } from "react";
import { useCameraStore } from "../stores/cameraStore";
import DiscoveryPanel from "../components/DiscoveryPanel";
import * as api from "../lib/api";
import type { CreateCameraRequest } from "../lib/types";

export default function Settings() {
  const { cameras, fetchCameras } = useCameraStore();
  const [showAddForm, setShowAddForm] = useState(false);

  useEffect(() => {
    fetchCameras();
  }, [fetchCameras]);

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Delete camera "${name}"?`)) return;
    try {
      await api.deleteCamera(id);
      fetchCameras();
    } catch (e) {
      alert((e as Error).message);
    }
  };

  const handleToggle = async (id: string, enabled: boolean) => {
    try {
      await api.updateCamera(id, { enabled: !enabled });
      fetchCameras();
    } catch (e) {
      alert((e as Error).message);
    }
  };

  return (
    <div className="p-4 max-w-5xl mx-auto space-y-6">
      <h2 className="text-xl font-semibold">Settings</h2>

      {/* Discovery */}
      <DiscoveryPanel onCamerasAdded={fetchCameras} />

      {/* Camera list */}
      <div className="bg-gray-900 rounded-lg p-4">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">
            Cameras ({cameras.length})
          </h3>
          <button
            onClick={() => setShowAddForm(!showAddForm)}
            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 rounded text-sm font-medium transition-colors"
          >
            {showAddForm ? "Cancel" : "Add Manual"}
          </button>
        </div>

        {showAddForm && (
          <AddCameraForm
            onAdded={() => {
              fetchCameras();
              setShowAddForm(false);
            }}
          />
        )}

        {cameras.length === 0 ? (
          <p className="text-gray-400 text-sm">
            No cameras. Use ONVIF Discovery above or add manually.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-gray-400 border-b border-gray-800">
                  <th className="pb-2 pr-4">Name</th>
                  <th className="pb-2 pr-4">RTSP Main</th>
                  <th className="pb-2 pr-4">Status</th>
                  <th className="pb-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {cameras.map((cam) => (
                  <tr
                    key={cam.id}
                    className="border-b border-gray-800/50 hover:bg-gray-800/30"
                  >
                    <td className="py-2 pr-4 font-medium">{cam.name}</td>
                    <td className="py-2 pr-4 text-gray-400 truncate max-w-xs">
                      {cam.rtsp_main}
                    </td>
                    <td className="py-2 pr-4">
                      <button
                        onClick={() => handleToggle(cam.id, cam.enabled)}
                        className={`px-2 py-0.5 rounded text-xs font-medium ${
                          cam.enabled
                            ? "bg-green-900/50 text-green-400"
                            : "bg-red-900/50 text-red-400"
                        }`}
                      >
                        {cam.enabled ? "Enabled" : "Disabled"}
                      </button>
                    </td>
                    <td className="py-2">
                      <button
                        onClick={() => handleDelete(cam.id, cam.name)}
                        className="text-red-400 hover:text-red-300 text-xs"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

function AddCameraForm({ onAdded }: { onAdded: () => void }) {
  const [name, setName] = useState("");
  const [rtspMain, setRtspMain] = useState("");
  const [rtspSub, setRtspSub] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const data: CreateCameraRequest = {
        name,
        rtsp_main: rtspMain,
      };
      if (rtspSub) data.rtsp_sub = rtspSub;
      if (username) data.username = username;
      if (password) data.password = password;

      await api.createCamera(data);
      onAdded();
    } catch (e) {
      alert((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="mb-4 p-3 bg-gray-800 rounded-lg space-y-3">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <input
          type="text"
          placeholder="Camera name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          className="bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        />
        <input
          type="text"
          placeholder="RTSP main URL"
          value={rtspMain}
          onChange={(e) => setRtspMain(e.target.value)}
          required
          className="bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        />
        <input
          type="text"
          placeholder="RTSP sub URL (optional)"
          value={rtspSub}
          onChange={(e) => setRtspSub(e.target.value)}
          className="bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        />
        <input
          type="text"
          placeholder="Username (optional)"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          className="bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        />
        <input
          type="password"
          placeholder="Password (optional)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="bg-gray-900 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        />
      </div>
      <button
        type="submit"
        disabled={saving || !name || !rtspMain}
        className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 rounded text-sm font-medium transition-colors"
      >
        {saving ? "Adding..." : "Add Camera"}
      </button>
    </form>
  );
}

import { useState } from "react";
import type { DiscoveredCamera } from "../lib/types";
import * as api from "../lib/api";

export default function DiscoveryPanel({
  onCamerasAdded,
}: {
  onCamerasAdded: () => void;
}) {
  const [address, setAddress] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [discovered, setDiscovered] = useState<DiscoveredCamera[]>([]);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [scanning, setScanning] = useState(false);
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleScan = async () => {
    setScanning(true);
    setError(null);
    setDiscovered([]);
    setSelected(new Set());
    try {
      const cameras = await api.discoverCameras({
        address,
        username,
        password,
      });
      setDiscovered(cameras);
      setSelected(new Set(cameras.map((_, i) => i)));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setScanning(false);
    }
  };

  const toggleSelect = (idx: number) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(idx)) next.delete(idx);
      else next.add(idx);
      return next;
    });
  };

  const toggleAll = () => {
    if (selected.size === discovered.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(discovered.map((_, i) => i)));
    }
  };

  const handleAdd = async () => {
    setAdding(true);
    setError(null);
    try {
      const cameras = discovered
        .filter((_, i) => selected.has(i))
        .map((c) => ({
          name: c.name,
          rtsp_main: c.rtsp_main,
          rtsp_sub: c.rtsp_sub,
        }));

      await api.addDiscoveredCameras({
        cameras,
        username: username || undefined,
        password: password || undefined,
      });

      setDiscovered([]);
      setSelected(new Set());
      onCamerasAdded();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="bg-gray-900 rounded-lg p-4">
      <h3 className="text-lg font-semibold mb-4">ONVIF Discovery</h3>

      {/* Scan form */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-4">
        <input
          type="text"
          placeholder="DVR address (e.g., 192.168.1.160)"
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          className="bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        />
        <input
          type="text"
          placeholder="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          className="bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        />
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        />
      </div>

      <button
        onClick={handleScan}
        disabled={scanning || !address}
        className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 rounded text-sm font-medium transition-colors"
      >
        {scanning ? "Scanning..." : "Scan Network"}
      </button>

      {error && (
        <p className="mt-3 text-sm text-red-400">Error: {error}</p>
      )}

      {/* Discovered cameras */}
      {discovered.length > 0 && (
        <div className="mt-4">
          <div className="flex items-center justify-between mb-2">
            <p className="text-sm text-gray-400">
              Found {discovered.length} camera(s)
            </p>
            <button
              onClick={toggleAll}
              className="text-sm text-blue-400 hover:text-blue-300"
            >
              {selected.size === discovered.length
                ? "Deselect All"
                : "Select All"}
            </button>
          </div>

          <div className="space-y-1 max-h-64 overflow-y-auto">
            {discovered.map((cam, i) => (
              <label
                key={i}
                className="flex items-center gap-3 px-3 py-2 rounded bg-gray-800 hover:bg-gray-750 cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={selected.has(i)}
                  onChange={() => toggleSelect(i)}
                  className="rounded"
                />
                <span className="text-sm flex-1">
                  Ch{cam.channel}: {cam.name}
                </span>
                <span className="text-xs text-gray-500 truncate max-w-xs">
                  {cam.rtsp_main}
                </span>
              </label>
            ))}
          </div>

          <button
            onClick={handleAdd}
            disabled={adding || selected.size === 0}
            className="mt-3 px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 rounded text-sm font-medium transition-colors"
          >
            {adding
              ? "Adding..."
              : `Add ${selected.size} Camera(s)`}
          </button>
        </div>
      )}
    </div>
  );
}

import { useState } from "react";
import WatchlistPanel from "../components/faces/WatchlistPanel";
import AlertsPanel from "../components/faces/AlertsPanel";
import VisitorsPanel from "../components/faces/VisitorsPanel";
import FaceConfigPanel from "../components/faces/FaceConfigPanel";

type Tab = "watchlist" | "alerts" | "visitors" | "settings";

const tabs: { key: Tab; label: string }[] = [
  { key: "watchlist", label: "Watchlist" },
  { key: "alerts", label: "Alerts" },
  { key: "visitors", label: "Visitors" },
  { key: "settings", label: "Settings" },
];

export default function Faces() {
  const [tab, setTab] = useState<Tab>("watchlist");

  return (
    <div className="p-4">
      <div className="flex gap-1 mb-4">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-3 py-1.5 rounded text-sm font-medium transition-colors ${
              tab === t.key
                ? "bg-blue-600 text-white"
                : "text-gray-400 hover:text-gray-200 hover:bg-gray-800"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === "watchlist" && <WatchlistPanel />}
      {tab === "alerts" && <AlertsPanel />}
      {tab === "visitors" && <VisitorsPanel />}
      {tab === "settings" && <FaceConfigPanel />}
    </div>
  );
}

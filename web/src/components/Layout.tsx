import { type ReactNode } from "react";
import { useCameraStore } from "../stores/cameraStore";
import type { Page } from "../App";

export default function Layout({
  children,
  currentPage,
  onNavigate,
}: {
  children: ReactNode;
  currentPage: Page;
  onNavigate: (page: Page) => void;
}) {
  const { viewMode, setViewMode, appMode, setAppMode } = useCameraStore();

  return (
    <div className="min-h-screen flex flex-col">
      <header className="bg-gray-900 border-b border-gray-800 px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-lg font-semibold tracking-tight">Surveillance</h1>
          {currentPage === "live" && (
            <>
              <div className="flex gap-0.5 bg-gray-800 rounded p-0.5">
                <button
                  onClick={() => setAppMode("live")}
                  className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                    appMode === "live"
                      ? "bg-blue-600 text-white"
                      : "text-gray-400 hover:text-gray-200"
                  }`}
                >
                  Live
                </button>
                <button
                  onClick={() => setAppMode("playback")}
                  className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                    appMode === "playback"
                      ? "bg-blue-600 text-white"
                      : "text-gray-400 hover:text-gray-200"
                  }`}
                >
                  Playback
                </button>
              </div>
              {appMode === "live" && (
                <div className="flex gap-0.5 bg-gray-800 rounded p-0.5">
                  <button
                    onClick={() => setViewMode("single")}
                    className={`px-2 py-1 rounded text-xs font-bold transition-colors ${
                      viewMode === "single"
                        ? "bg-blue-600 text-white"
                        : "text-gray-400 hover:text-gray-200"
                    }`}
                  >
                    1
                  </button>
                  <button
                    onClick={() => setViewMode("quad")}
                    className={`px-2 py-1 rounded text-xs font-bold transition-colors ${
                      viewMode === "quad"
                        ? "bg-blue-600 text-white"
                        : "text-gray-400 hover:text-gray-200"
                    }`}
                  >
                    4
                  </button>
                </div>
              )}
            </>
          )}
        </div>
        <nav className="flex gap-1">
          {(
            [
              { key: "live" as Page, label: "Live View" },
              { key: "faces" as Page, label: "Faces" },
              { key: "settings" as Page, label: "Settings" },
            ] as const
          ).map((item) => (
            <button
              key={item.key}
              onClick={() => onNavigate(item.key)}
              className={`px-3 py-1.5 rounded text-sm font-medium transition-colors ${
                currentPage === item.key
                  ? "bg-blue-600 text-white"
                  : "text-gray-400 hover:text-gray-200 hover:bg-gray-800"
              }`}
            >
              {item.label}
            </button>
          ))}
        </nav>
      </header>
      <main className="flex-1">{children}</main>
    </div>
  );
}

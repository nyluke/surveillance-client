import { create } from "zustand";
import type { Camera, Group } from "../lib/types";
import * as api from "../lib/api";

type ViewMode = "single" | "quad";
type AppMode = "live" | "playback";

interface CameraState {
  cameras: Camera[];
  groups: Group[];
  loading: boolean;
  error: string | null;
  viewMode: ViewMode;
  selectedCamera: string | null;
  quadCameras: (string | null)[];

  // Playback state
  appMode: AppMode;
  playbackDate: string | null;
  playbackTime: string | null;
  playbackDuration: number;
  playbackSpeed: number;
  playbackActive: boolean;

  fetchCameras: () => Promise<void>;
  fetchGroups: () => Promise<void>;
  deleteCamera: (id: string) => Promise<void>;
  setViewMode: (mode: ViewMode) => void;
  selectCamera: (id: string) => void;
  assignQuad: (slot: number, id: string) => void;
  clearQuadSlot: (slot: number) => void;

  // Playback actions
  setAppMode: (mode: AppMode) => void;
  setPlaybackDate: (date: string) => void;
  setPlaybackTime: (time: string) => void;
  setPlaybackDuration: (duration: number) => void;
  setPlaybackSpeed: (speed: number) => void;
  setPlaybackActive: (active: boolean) => void;
  stopPlayback: () => void;
}

export const useCameraStore = create<CameraState>((set, get) => ({
  cameras: [],
  groups: [],
  loading: false,
  error: null,
  viewMode: "single",
  selectedCamera: null,
  quadCameras: [null, null, null, null],

  // Playback state
  appMode: "live",
  playbackDate: null,
  playbackTime: null,
  playbackDuration: 3600,
  playbackSpeed: 1,
  playbackActive: false,

  fetchCameras: async () => {
    set({ loading: true, error: null });
    try {
      const cameras = await api.listCameras();
      set({ cameras, loading: false });
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  fetchGroups: async () => {
    try {
      const groups = await api.listGroups();
      set({ groups });
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  deleteCamera: async (id) => {
    try {
      await api.deleteCamera(id);
      const cameras = get().cameras.filter((c) => c.id !== id);
      set({ cameras });
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  setViewMode: (mode) => {
    const { selectedCamera, quadCameras } = get();
    if (mode === "quad" && selectedCamera && !quadCameras.includes(selectedCamera)) {
      set({ viewMode: mode, quadCameras: [selectedCamera, null, null, null] });
    } else if (mode === "single" && !selectedCamera) {
      const first = quadCameras.find((id) => id !== null) ?? null;
      set({ viewMode: mode, selectedCamera: first });
    } else {
      set({ viewMode: mode });
    }
  },

  selectCamera: (id) => {
    const { viewMode, quadCameras, appMode } = get();

    if (appMode === "playback") {
      set({ selectedCamera: id, playbackActive: false, playbackSpeed: 1 });
      return;
    }

    if (viewMode === "single") {
      set({ selectedCamera: id });
    } else {
      const idx = quadCameras.indexOf(id);
      if (idx !== -1) return;
      const emptyIdx = quadCameras.indexOf(null);
      if (emptyIdx !== -1) {
        const next = [...quadCameras];
        next[emptyIdx] = id;
        set({ quadCameras: next });
      } else {
        set({ quadCameras: [id, quadCameras[1], quadCameras[2], quadCameras[3]] });
      }
    }
  },

  assignQuad: (slot, id) => {
    const next = [...get().quadCameras];
    next[slot] = id;
    set({ quadCameras: next });
  },

  clearQuadSlot: (slot) => {
    const next = [...get().quadCameras];
    next[slot] = null;
    set({ quadCameras: next });
  },

  // Playback actions

  setAppMode: (mode) => {
    if (mode === "live") {
      set({ playbackActive: false, playbackSpeed: 1 });
    }
    set({ appMode: mode });
  },

  setPlaybackDate: (date) => set({ playbackDate: date }),
  setPlaybackTime: (time) => set({ playbackTime: time }),
  setPlaybackDuration: (duration) => set({ playbackDuration: duration }),
  setPlaybackSpeed: (speed) => set({ playbackSpeed: speed }),
  setPlaybackActive: (active) => set({ playbackActive: active }),

  stopPlayback: () => {
    set({ playbackActive: false, playbackSpeed: 1 });
  },
}));

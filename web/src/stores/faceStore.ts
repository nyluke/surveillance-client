import { create } from "zustand";
import type {
  FaceSubject,
  FaceSighting,
  FaceCluster,
  FaceMonitorConfig,
  FaceServiceStatus,
} from "../lib/types";
import * as api from "../lib/api";

interface FaceState {
  subjects: FaceSubject[];
  sightings: FaceSighting[];
  clusters: FaceCluster[];
  configs: FaceMonitorConfig[];
  status: FaceServiceStatus | null;
  loading: boolean;
  error: string | null;

  fetchSubjects: () => Promise<void>;
  createSubject: (formData: FormData) => Promise<void>;
  deleteSubject: (id: string) => Promise<void>;
  fetchSightings: (limit?: number) => Promise<void>;
  fetchClusters: () => Promise<void>;
  labelCluster: (id: string, label: string) => Promise<void>;
  triggerClustering: () => Promise<void>;
  fetchConfig: () => Promise<void>;
  saveConfig: (configs: FaceMonitorConfig[]) => Promise<void>;
  fetchStatus: () => Promise<void>;
  clearError: () => void;
}

export const useFaceStore = create<FaceState>((set) => ({
  subjects: [],
  sightings: [],
  clusters: [],
  configs: [],
  status: null,
  loading: false,
  error: null,

  fetchSubjects: async () => {
    set({ loading: true, error: null });
    try {
      const subjects = await api.listFaceSubjects();
      set({ subjects, loading: false });
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  createSubject: async (formData) => {
    set({ loading: true, error: null });
    try {
      await api.createFaceSubject(formData);
      const subjects = await api.listFaceSubjects();
      set({ subjects, loading: false });
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  deleteSubject: async (id) => {
    try {
      await api.deleteFaceSubject(id);
      set((s) => ({ subjects: s.subjects.filter((sub) => sub.id !== id) }));
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  fetchSightings: async (limit = 100) => {
    try {
      const sightings = await api.listFaceSightings(limit);
      set({ sightings });
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  fetchClusters: async () => {
    try {
      const clusters = await api.listFaceClusters();
      set({ clusters });
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  labelCluster: async (id, label) => {
    try {
      await api.labelFaceCluster(id, label);
      set((s) => ({
        clusters: s.clusters.map((c) => (c.id === id ? { ...c, label } : c)),
      }));
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  triggerClustering: async () => {
    set({ loading: true, error: null });
    try {
      await api.triggerFaceClustering();
      const clusters = await api.listFaceClusters();
      set({ clusters, loading: false });
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  fetchConfig: async () => {
    try {
      const configs = await api.getFaceConfig();
      set({ configs });
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  saveConfig: async (configs) => {
    try {
      await api.setFaceConfig(configs);
      set({ configs });
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  fetchStatus: async () => {
    try {
      const status = await api.getFaceServiceStatus();
      set({ status });
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  clearError: () => set({ error: null }),
}));

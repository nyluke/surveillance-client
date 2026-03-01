import type {
  Camera,
  Group,
  CreateCameraRequest,
  UpdateCameraRequest,
  DiscoverRequest,
  DiscoveredCamera,
  AddDiscoveredRequest,
} from "./types";

const BASE = "/api";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
  return res.json();
}

// Cameras
export const listCameras = () => request<Camera[]>("/cameras");
export const getCamera = (id: string) => request<Camera>(`/cameras/${id}`);
export const createCamera = (data: CreateCameraRequest) =>
  request<Camera>("/cameras", { method: "POST", body: JSON.stringify(data) });
export const updateCamera = (id: string, data: UpdateCameraRequest) =>
  request<Camera>(`/cameras/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
export const deleteCamera = (id: string) =>
  request<{ status: string }>(`/cameras/${id}`, { method: "DELETE" });

// Groups
export const listGroups = () => request<Group[]>("/groups");
export const createGroup = (data: { name: string; sort_order?: number }) =>
  request<Group>("/groups", { method: "POST", body: JSON.stringify(data) });
export const updateGroup = (
  id: string,
  data: { name?: string; sort_order?: number },
) =>
  request<Group>(`/groups/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
export const deleteGroup = (id: string) =>
  request<{ status: string }>(`/groups/${id}`, { method: "DELETE" });

// Discovery
export const discoverCameras = (data: DiscoverRequest) =>
  request<DiscoveredCamera[]>("/discover", {
    method: "POST",
    body: JSON.stringify(data),
  });
export const addDiscoveredCameras = (data: AddDiscoveredRequest) =>
  request<Camera[]>("/discover/add", {
    method: "POST",
    body: JSON.stringify(data),
  });


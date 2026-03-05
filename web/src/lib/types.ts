export interface Camera {
  id: string;
  name: string;
  onvif_address?: string;
  rtsp_main: string;
  rtsp_sub?: string;
  username?: string;
  password?: string;
  enabled: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface Group {
  id: string;
  name: string;
  sort_order: number;
}

export interface CreateCameraRequest {
  name: string;
  rtsp_main: string;
  rtsp_sub?: string;
  onvif_address?: string;
  username?: string;
  password?: string;
  enabled?: boolean;
  sort_order?: number;
}

export interface UpdateCameraRequest {
  name?: string;
  rtsp_main?: string;
  rtsp_sub?: string;
  onvif_address?: string;
  username?: string;
  password?: string;
  enabled?: boolean;
  sort_order?: number;
}

export interface DiscoveredCamera {
  channel: number;
  name: string;
  rtsp_main: string;
  rtsp_sub?: string;
  profile_id: string;
}

export interface DiscoverRequest {
  address: string;
  username: string;
  password: string;
}

export interface AddDiscoveredRequest {
  cameras: { name: string; rtsp_main: string; rtsp_sub?: string }[];
  username?: string;
  password?: string;
}

// Face recognition types

export interface FaceSubject {
  id: string;
  name: string;
  notes?: string;
  alert_enabled: boolean;
  created_at: string;
  crop_url?: string;
}

export interface FaceSighting {
  id: string;
  subject_id: string;
  camera_id: string;
  confidence: number;
  crop_path?: string;
  seen_at: string;
  subject_name: string;
  camera_name: string;
  crop_url?: string;
}

export interface FaceCluster {
  id: string;
  label?: string;
  first_seen: string;
  last_seen: string;
  visit_count: number;
  representative_crop?: string;
}

export interface FaceMonitorConfig {
  camera_id: string;
  monitor_type: "realtime" | "batch" | "both";
  interval_seconds: number;
}

export interface FaceServiceStatus {
  face_service_configured: boolean;
  face_service_online?: boolean;
  face_service_error?: string;
  slack_configured: boolean;
}


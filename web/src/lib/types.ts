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


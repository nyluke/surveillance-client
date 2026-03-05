package face

type Subject struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Notes        string `json:"notes,omitempty"`
	AlertEnabled bool   `json:"alert_enabled"`
	CreatedAt    string `json:"created_at"`
}

type Embedding struct {
	ID        string  `json:"id"`
	SubjectID string  `json:"subject_id"`
	Embedding []byte  `json:"embedding,omitempty"`
	CropPath  string  `json:"crop_path,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type Sighting struct {
	ID         string  `json:"id"`
	SubjectID  string  `json:"subject_id"`
	CameraID   string  `json:"camera_id"`
	Confidence float64 `json:"confidence"`
	CropPath   string  `json:"crop_path,omitempty"`
	SeenAt     string  `json:"seen_at"`
}

type Visitor struct {
	ID        string `json:"id"`
	ClusterID string `json:"cluster_id,omitempty"`
	CameraID  string `json:"camera_id"`
	Embedding []byte `json:"embedding,omitempty"`
	CropPath  string `json:"crop_path,omitempty"`
	SeenAt    string `json:"seen_at"`
}

type Cluster struct {
	ID                 string `json:"id"`
	Label              string `json:"label,omitempty"`
	FirstSeen          string `json:"first_seen"`
	LastSeen           string `json:"last_seen"`
	VisitCount         int    `json:"visit_count"`
	RepresentativeCrop string `json:"representative_crop,omitempty"`
}

type MonitorConfig struct {
	CameraID        string `json:"camera_id"`
	MonitorType     string `json:"monitor_type"`
	IntervalSeconds int    `json:"interval_seconds"`
	RTSPUrl         string `json:"rtsp_url,omitempty"`
	CameraName      string `json:"camera_name,omitempty"`
}

type Alert struct {
	ID           string `json:"id"`
	SightingID   string `json:"sighting_id"`
	SentAt       string `json:"sent_at"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// API request/response types

type CreateSubjectRequest struct {
	Name  string `json:"name"`
	Notes string `json:"notes,omitempty"`
}

type EnrollResponse struct {
	Embedding []float64 `json:"embedding"`
	CropB64   string    `json:"crop_base64"`
}

type ReportSightingRequest struct {
	SubjectID  string  `json:"subject_id"`
	CameraID   string  `json:"camera_id"`
	Confidence float64 `json:"confidence"`
	CropB64    string  `json:"crop_base64,omitempty"`
}

type ReportVisitorRequest struct {
	CameraID  string    `json:"camera_id"`
	Embedding []float64 `json:"embedding"`
	CropB64   string    `json:"crop_base64,omitempty"`
}

type GalleryEntry struct {
	SubjectID    string    `json:"subject_id"`
	EmbeddingID  string    `json:"embedding_id"`
	Name         string    `json:"name"`
	AlertEnabled bool      `json:"alert_enabled"`
	Embedding    []float64 `json:"embedding"`
}

type ClusterRequest struct {
	Embeddings [][]float64 `json:"embeddings"`
	IDs        []string    `json:"ids"`
}

type ClusterResponse struct {
	Labels []int `json:"labels"`
}

type SubjectWithCrop struct {
	Subject
	CropURL string `json:"crop_url,omitempty"`
}

type SightingWithDetails struct {
	Sighting
	SubjectName string `json:"subject_name"`
	CameraName  string `json:"camera_name"`
	CropURL     string `json:"crop_url,omitempty"`
}

type UpdateMonitorConfigRequest struct {
	Configs []MonitorConfig `json:"configs"`
}

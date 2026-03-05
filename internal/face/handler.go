package face

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// CameraLister provides camera info for RTSP snapshot URLs
type CameraLister interface {
	GetCamera(id string) (CameraInfo, bool)
}

// CameraInfo is the subset of camera data needed by the face handler
type CameraInfo struct {
	RTSPMain string
	RTSPSub  *string
	Username *string
	Password *string
	Name     string
}

type Handler struct {
	store          *Store
	alerter        *Alerter
	snapshot       *SnapshotFetcher
	cameras        CameraLister
	faceServiceURL string
	faceDataDir    string
}

func NewHandler(store *Store, alerter *Alerter, snapshot *SnapshotFetcher, cameras CameraLister, faceServiceURL, faceDataDir string) *Handler {
	if faceDataDir != "" {
		os.MkdirAll(faceDataDir, 0755)
	}
	return &Handler{
		store:          store,
		alerter:        alerter,
		snapshot:       snapshot,
		cameras:        cameras,
		faceServiceURL: faceServiceURL,
		faceDataDir:    faceDataDir,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/faces")
	path = strings.TrimPrefix(path, "/")

	switch {
	// Subject routes
	case path == "subjects" && r.Method == http.MethodGet:
		h.listSubjects(w, r)
	case path == "subjects" && r.Method == http.MethodPost:
		h.createSubject(w, r)
	case strings.HasPrefix(path, "subjects/") && r.Method == http.MethodDelete:
		id := strings.TrimPrefix(path, "subjects/")
		h.deleteSubject(w, r, id)

	// Crop images
	case strings.HasPrefix(path, "crops/") && r.Method == http.MethodGet:
		filename := strings.TrimPrefix(path, "crops/")
		h.serveCrop(w, r, filename)

	// Sightings
	case path == "sightings" && r.Method == http.MethodGet:
		h.listSightings(w, r)

	// Visitors / clusters
	case path == "visitors" && r.Method == http.MethodGet:
		h.listClusters(w, r)
	case strings.HasPrefix(path, "visitors/cluster") && r.Method == http.MethodPost:
		h.triggerClustering(w, r)
	case strings.HasPrefix(path, "visitors/") && r.Method == http.MethodPut:
		clusterID := strings.TrimPrefix(path, "visitors/")
		h.labelCluster(w, r, clusterID)

	// Monitor config
	case path == "config" && r.Method == http.MethodGet:
		h.getConfig(w, r)
	case path == "config" && r.Method == http.MethodPut:
		h.setConfig(w, r)

	// Snapshot proxy
	case strings.HasPrefix(path, "snapshots/") && r.Method == http.MethodGet:
		cameraID := strings.TrimPrefix(path, "snapshots/")
		h.proxySnapshot(w, r, cameraID)

	// Internal endpoints (called by Python service)
	case path == "internal/sighting" && r.Method == http.MethodPost:
		h.reportSighting(w, r)
	case path == "internal/visitor" && r.Method == http.MethodPost:
		h.reportVisitor(w, r)
	case path == "internal/gallery" && r.Method == http.MethodGet:
		h.getGallery(w, r)

	// Service status
	case path == "status" && r.Method == http.MethodGet:
		h.serviceStatus(w, r)

	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// Subject endpoints

func (h *Handler) listSubjects(w http.ResponseWriter, _ *http.Request) {
	subjects, err := h.store.ListSubjects()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Enrich with crop URLs
	result := make([]SubjectWithCrop, len(subjects))
	for i, sub := range subjects {
		result[i] = SubjectWithCrop{Subject: sub}
		crop, _ := h.store.GetFirstEmbeddingCrop(sub.ID)
		if crop != "" {
			result[i].CropURL = "/api/faces/crops/" + filepath.Base(crop)
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) createSubject(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (image + name + notes)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	name := r.FormValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	notes := r.FormValue("notes")

	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read image")
		return
	}

	// Forward to Python face service for enrollment
	enrollResp, err := h.enrollWithService(imageData)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("face service error: %v", err))
		return
	}

	// Create subject
	subject, err := h.store.CreateSubject(name, notes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Save crop image
	cropFilename := ""
	if enrollResp.CropB64 != "" {
		cropData, err := base64.StdEncoding.DecodeString(enrollResp.CropB64)
		if err == nil {
			cropFilename = fmt.Sprintf("%s.jpg", uuid.New().String())
			cropPath := filepath.Join(h.faceDataDir, cropFilename)
			os.WriteFile(cropPath, cropData, 0644)
		}
	}

	// Store embedding
	embBytes := EncodeEmbedding(enrollResp.Embedding)
	_, err = h.store.AddEmbedding(subject.ID, embBytes, cropFilename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := SubjectWithCrop{Subject: *subject}
	if cropFilename != "" {
		result.CropURL = "/api/faces/crops/" + cropFilename
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) enrollWithService(imageData []byte) (*EnrollResponse, error) {
	if h.faceServiceURL == "" {
		return nil, fmt.Errorf("face service not configured")
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("image", "photo.jpg")
	if err != nil {
		return nil, err
	}
	part.Write(imageData)
	writer.Close()

	resp, err := http.Post(h.faceServiceURL+"/enroll", writer.FormDataContentType(), &buf)
	if err != nil {
		return nil, fmt.Errorf("face service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("face service returned %d: %s", resp.StatusCode, string(body))
	}

	var result EnrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid face service response: %w", err)
	}
	return &result, nil
}

func (h *Handler) deleteSubject(w http.ResponseWriter, _ *http.Request, id string) {
	if err := h.store.DeleteSubject(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Crop serving

func (h *Handler) serveCrop(w http.ResponseWriter, r *http.Request, filename string) {
	// Sanitize filename to prevent path traversal
	filename = filepath.Base(filename)
	path := filepath.Join(h.faceDataDir, filename)
	http.ServeFile(w, r, path)
}

// Sighting endpoints

func (h *Handler) listSightings(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	sightings, err := h.store.ListSightings(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for i := range sightings {
		if sightings[i].CropPath != "" {
			sightings[i].CropURL = "/api/faces/crops/" + filepath.Base(sightings[i].CropPath)
		}
	}

	writeJSON(w, http.StatusOK, sightings)
}

// Visitor / Cluster endpoints

func (h *Handler) listClusters(w http.ResponseWriter, _ *http.Request) {
	clusters, err := h.store.ListClusters()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	for i := range clusters {
		if clusters[i].RepresentativeCrop != "" {
			clusters[i].RepresentativeCrop = "/api/faces/crops/" + filepath.Base(clusters[i].RepresentativeCrop)
		}
	}

	writeJSON(w, http.StatusOK, clusters)
}

func (h *Handler) labelCluster(w http.ResponseWriter, r *http.Request, clusterID string) {
	var req struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.store.UpdateClusterLabel(clusterID, req.Label); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) triggerClustering(w http.ResponseWriter, _ *http.Request) {
	if h.faceServiceURL == "" {
		writeError(w, http.StatusBadGateway, "face service not configured")
		return
	}

	visitors, err := h.store.ListVisitorsForClustering()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(visitors) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{"status": "no visitors to cluster"})
		return
	}

	// Prepare embeddings for clustering
	embeddings := make([][]float64, len(visitors))
	ids := make([]string, len(visitors))
	for i, v := range visitors {
		embeddings[i] = DecodeEmbedding(v.Embedding)
		ids[i] = v.ID
	}

	req := ClusterRequest{Embeddings: embeddings, IDs: ids}
	body, _ := json.Marshal(req)

	resp, err := http.Post(h.faceServiceURL+"/cluster", "application/json", bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("face service error: %v", err))
		return
	}
	defer resp.Body.Close()

	var clusterResp ClusterResponse
	if err := json.NewDecoder(resp.Body).Decode(&clusterResp); err != nil {
		writeError(w, http.StatusBadGateway, "invalid cluster response")
		return
	}

	// Clear existing clusters and rebuild
	h.store.ClearClusters()

	// Group visitors by cluster label
	clusterMap := make(map[int][]int) // cluster_label -> visitor indices
	for i, label := range clusterResp.Labels {
		if label >= 0 { // -1 = noise in DBSCAN
			clusterMap[label] = append(clusterMap[label], i)
		}
	}

	for label, indices := range clusterMap {
		clusterID := fmt.Sprintf("cluster_%d", label)
		firstSeen := visitors[indices[0]].SeenAt
		lastSeen := visitors[indices[0]].SeenAt
		repCrop := visitors[indices[0]].CropPath

		for _, idx := range indices {
			v := visitors[idx]
			if v.SeenAt < firstSeen {
				firstSeen = v.SeenAt
			}
			if v.SeenAt > lastSeen {
				lastSeen = v.SeenAt
			}
			h.store.UpdateVisitorCluster(v.ID, clusterID)
		}

		h.store.UpsertCluster(clusterID, "", firstSeen, lastSeen, len(indices), repCrop)
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "clustered", "clusters": len(clusterMap)})
}

// Monitor config endpoints

func (h *Handler) getConfig(w http.ResponseWriter, _ *http.Request) {
	configs, err := h.store.ListMonitorConfigs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Enrich with RTSP URLs for Python service (main stream for face detection resolution)
	if h.cameras != nil {
		for i := range configs {
			if cam, ok := h.cameras.GetCamera(configs[i].CameraID); ok {
				configs[i].CameraName = cam.Name
				configs[i].RTSPUrl = injectRTSPCredentials(cam.RTSPMain, cam.Username, cam.Password)
			}
		}
	}
	writeJSON(w, http.StatusOK, configs)
}

func (h *Handler) setConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateMonitorConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.store.SetMonitorConfigs(req.Configs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Snapshot proxy

func (h *Handler) proxySnapshot(w http.ResponseWriter, _ *http.Request, cameraID string) {
	// Look up camera RTSP URL (use main stream for face detection resolution)
	var rtspURL string
	if h.cameras != nil {
		if cam, ok := h.cameras.GetCamera(cameraID); ok {
			rtspURL = cam.RTSPMain
			rtspURL = injectRTSPCredentials(rtspURL, cam.Username, cam.Password)
		}
	}
	if rtspURL == "" {
		writeError(w, http.StatusNotFound, "camera not found or no RTSP URL")
		return
	}

	data, err := h.snapshot.FetchJPEGFromRTSP(rtspURL)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

// Internal endpoints (called by Python service)

func (h *Handler) reportSighting(w http.ResponseWriter, r *http.Request) {
	var req ReportSightingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Save crop if provided
	cropFilename := ""
	if req.CropB64 != "" {
		cropData, err := base64.StdEncoding.DecodeString(req.CropB64)
		if err == nil {
			cropFilename = fmt.Sprintf("sighting_%s.jpg", uuid.New().String())
			cropPath := filepath.Join(h.faceDataDir, cropFilename)
			os.WriteFile(cropPath, cropData, 0644)
		}
	}

	sighting, err := h.store.CreateSighting(req.SubjectID, req.CameraID, req.Confidence, cropFilename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Send Slack alert
	if h.alerter != nil && h.alerter.Enabled() {
		subject, _ := h.store.GetSubject(req.SubjectID)
		subjectName := "Unknown"
		if subject != nil {
			subjectName = subject.Name
		}
		// Look up camera name from sighting's camera_id
		go h.alerter.SendAlert(sighting, subjectName, req.CameraID)
	}

	writeJSON(w, http.StatusCreated, sighting)
}

func (h *Handler) reportVisitor(w http.ResponseWriter, r *http.Request) {
	var req ReportVisitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	cropFilename := ""
	if req.CropB64 != "" {
		cropData, err := base64.StdEncoding.DecodeString(req.CropB64)
		if err == nil {
			cropFilename = fmt.Sprintf("visitor_%s.jpg", uuid.New().String())
			cropPath := filepath.Join(h.faceDataDir, cropFilename)
			os.WriteFile(cropPath, cropData, 0644)
		}
	}

	embBytes := EncodeEmbedding(req.Embedding)
	visitor, err := h.store.CreateVisitor(req.CameraID, embBytes, cropFilename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, visitor)
}

func (h *Handler) getGallery(w http.ResponseWriter, _ *http.Request) {
	gallery, err := h.store.GetGallery()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gallery)
}

// Service status

func (h *Handler) serviceStatus(w http.ResponseWriter, _ *http.Request) {
	status := map[string]any{
		"face_service_configured": h.faceServiceURL != "",
		"slack_configured":        h.alerter != nil && h.alerter.Enabled(),
	}

	if h.faceServiceURL != "" {
		resp, err := http.Get(h.faceServiceURL + "/health")
		if err != nil {
			status["face_service_online"] = false
			status["face_service_error"] = err.Error()
		} else {
			resp.Body.Close()
			status["face_service_online"] = resp.StatusCode == 200
		}
	}

	writeJSON(w, http.StatusOK, status)
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// injectRTSPCredentials adds user:pass@ to an RTSP URL if credentials are provided.
func injectRTSPCredentials(rtspURL string, username, password *string) string {
	if username == nil || *username == "" {
		return rtspURL
	}
	u, err := url.Parse(rtspURL)
	if err != nil {
		return rtspURL
	}
	pass := ""
	if password != nil {
		pass = *password
	}
	u.User = url.UserPassword(*username, pass)
	return u.String()
}


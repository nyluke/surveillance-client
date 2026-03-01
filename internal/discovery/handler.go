package discovery

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"surveillance-client/internal/camera"
	"surveillance-client/internal/config"
)

type Handler struct {
	cfg     *config.Config
	store   *camera.Store
	syncer  camera.StreamSyncer
}

func NewHandler(cfg *config.Config, store *camera.Store, syncer camera.StreamSyncer) *Handler {
	return &Handler{cfg: cfg, store: store, syncer: syncer}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/discover")
	path = strings.TrimPrefix(path, "/")

	switch {
	case path == "" && r.Method == http.MethodPost:
		h.discover(w, r)
	case path == "add" && r.Method == http.MethodPost:
		h.addDiscovered(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

type DiscoverRequest struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) discover(w http.ResponseWriter, r *http.Request) {
	var req DiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Address == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "address is required"})
		return
	}

	// Ensure address has scheme
	address := req.Address
	if !strings.HasPrefix(address, "http") {
		address = "http://" + address
	}

	client := NewONVIFClient(address, req.Username, req.Password)

	dvrHost := h.cfg.DVRHost
	if dvrHost == "" {
		// Try to extract host from the provided address for rewriting
		dvrHost = extractHost(req.Address)
	}

	cameras, err := client.DiscoverCameras(dvrHost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	sort.Slice(cameras, func(i, j int) bool {
		return cameras[i].Channel < cameras[j].Channel
	})

	writeJSON(w, http.StatusOK, cameras)
}

type AddDiscoveredRequest struct {
	Cameras  []DiscoveredCameraAdd `json:"cameras"`
	Username string                `json:"username,omitempty"`
	Password string                `json:"password,omitempty"`
}

type DiscoveredCameraAdd struct {
	Name     string `json:"name"`
	RTSPMain string `json:"rtsp_main"`
	RTSPSub  string `json:"rtsp_sub,omitempty"`
}

func (h *Handler) addDiscovered(w http.ResponseWriter, r *http.Request) {
	var req AddDiscoveredRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	var created []camera.Camera
	for i, dc := range req.Cameras {
		createReq := camera.CreateCameraRequest{
			Name:     dc.Name,
			RTSPMain: dc.RTSPMain,
		}
		if dc.RTSPSub != "" {
			createReq.RTSPSub = &dc.RTSPSub
		}
		if req.Username != "" {
			createReq.Username = &req.Username
		}
		if req.Password != "" {
			createReq.Password = &req.Password
		}
		order := i
		createReq.SortOrder = &order

		cam, err := h.store.CreateCamera(createReq)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		created = append(created, *cam)
	}

	if h.syncer != nil {
		h.syncer.SyncStreams()
	}

	writeJSON(w, http.StatusCreated, created)
}

func extractHost(address string) string {
	address = strings.TrimPrefix(address, "http://")
	address = strings.TrimPrefix(address, "https://")
	if idx := strings.Index(address, ":"); idx >= 0 {
		return address[:idx]
	}
	if idx := strings.Index(address, "/"); idx >= 0 {
		return address[:idx]
	}
	return address
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

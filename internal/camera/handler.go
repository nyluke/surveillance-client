package camera

import (
	"encoding/json"
	"net/http"
	"strings"
)

// StreamSyncer is implemented by go2rtc.Manager
type StreamSyncer interface {
	SyncStreams() error
}

type Handler struct {
	store  *Store
	syncer StreamSyncer
}

func NewHandler(store *Store, syncer StreamSyncer) *Handler {
	return &Handler{store: store, syncer: syncer}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/cameras/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/cameras/")
	id = strings.TrimPrefix(id, "/api/cameras")
	id = strings.TrimPrefix(id, "/")

	if id == "" {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) list(w http.ResponseWriter, _ *http.Request) {
	cameras, err := h.store.ListCameras()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cameras)
}

func (h *Handler) get(w http.ResponseWriter, _ *http.Request, id string) {
	cam, err := h.store.GetCamera(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cam == nil {
		writeError(w, http.StatusNotFound, "camera not found")
		return
	}
	writeJSON(w, http.StatusOK, cam)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateCameraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" || req.RTSPMain == "" {
		writeError(w, http.StatusBadRequest, "name and rtsp_main are required")
		return
	}

	cam, err := h.store.CreateCamera(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if h.syncer != nil {
		h.syncer.SyncStreams()
	}

	writeJSON(w, http.StatusCreated, cam)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateCameraRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	cam, err := h.store.UpdateCamera(id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if h.syncer != nil {
		h.syncer.SyncStreams()
	}

	writeJSON(w, http.StatusOK, cam)
}

func (h *Handler) delete(w http.ResponseWriter, _ *http.Request, id string) {
	if err := h.store.DeleteCamera(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if h.syncer != nil {
		h.syncer.SyncStreams()
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GroupHandler handles /api/groups

type GroupHandler struct {
	store *Store
}

func NewGroupHandler(store *Store) *GroupHandler {
	return &GroupHandler{store: store}
}

func (h *GroupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/groups/")
	id = strings.TrimPrefix(id, "/api/groups")
	id = strings.TrimPrefix(id, "/")

	if id == "" {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *GroupHandler) list(w http.ResponseWriter, _ *http.Request) {
	groups, err := h.store.ListGroups()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) get(w http.ResponseWriter, _ *http.Request, id string) {
	group, err := h.store.GetGroup(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if group == nil {
		writeError(w, http.StatusNotFound, "group not found")
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (h *GroupHandler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	group, err := h.store.CreateGroup(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (h *GroupHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	group, err := h.store.UpdateGroup(id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (h *GroupHandler) delete(w http.ResponseWriter, _ *http.Request, id string) {
	if err := h.store.DeleteGroup(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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

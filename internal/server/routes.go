package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) routes() {
	// Health check
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// Camera routes
	if s.deps.CameraHandler != nil {
		s.mux.Handle("/api/cameras", s.deps.CameraHandler)
		s.mux.Handle("/api/cameras/", s.deps.CameraHandler)
	}

	// Group routes
	if s.deps.GroupHandler != nil {
		s.mux.Handle("/api/groups", s.deps.GroupHandler)
		s.mux.Handle("/api/groups/", s.deps.GroupHandler)
	}

	// Discovery routes
	if s.deps.DiscoveryHandler != nil {
		s.mux.Handle("/api/discover", s.deps.DiscoveryHandler)
		s.mux.Handle("/api/discover/", s.deps.DiscoveryHandler)
	}

	// DVR WebSocket proxy
	if s.deps.DvrProxyHandler != nil {
		s.mux.Handle("/api/dvr/ws", s.deps.DvrProxyHandler)
	}

	// Export remux (AVI → MP4)
	if s.deps.ExportHandler != nil {
		s.mux.Handle("POST /api/export/remux", s.deps.ExportHandler)
	}

	// Face recognition routes
	if s.deps.FaceHandler != nil {
		s.mux.Handle("/api/faces/", s.deps.FaceHandler)
		s.mux.Handle("/api/faces", s.deps.FaceHandler)
	}

	// go2rtc reverse proxy
	s.mux.Handle("/go2rtc/", s.go2rtcProxy())

	// SPA — must be last
	s.mux.Handle("/", s.spaHandler())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

package dvr

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"

	"surveillance-client/internal/camera"
	"surveillance-client/internal/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ProxyHandler struct {
	cfg           *config.Config
	cameraStore   *camera.Store
	authenticator *Authenticator
}

func NewProxyHandler(cfg *config.Config, cameraStore *camera.Store) *ProxyHandler {
	return &ProxyHandler{
		cfg:           cfg,
		cameraStore:   cameraStore,
		authenticator: NewAuthenticator(),
	}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cameraID := r.URL.Query().Get("camera_id")
	if cameraID == "" {
		http.Error(w, "camera_id is required", http.StatusBadRequest)
		return
	}

	cam, err := h.cameraStore.GetCamera(cameraID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get camera: %v", err), http.StatusInternalServerError)
		return
	}
	if cam == nil {
		http.Error(w, "camera not found", http.StatusNotFound)
		return
	}

	host := h.cfg.DVRHost
	if host == "" {
		host = extractHost(cam.RTSPMain)
	}
	if host == "" {
		http.Error(w, "cannot determine DVR host", http.StatusBadRequest)
		return
	}

	username := h.cfg.DVRUsername
	password := h.cfg.DVRPassword

	session, err := h.authenticator.GetSession(host, username, password)
	if err != nil {
		log.Printf("DVR auth failed for %s: %v", host, err)
		http.Error(w, fmt.Sprintf("DVR authentication failed: %v", err), http.StatusBadGateway)
		return
	}

	// Connect to DVR WebSocket
	dvrURL := fmt.Sprintf("ws://%s/requestWebsocketConnection?sessionID=%s", host, url.QueryEscape(session.ID))
	dvrConn, err := dialDVR(dvrURL, session)
	if err != nil {
		// Session might be stale — invalidate and retry once
		h.authenticator.Invalidate(host, username)
		session, err = h.authenticator.GetSession(host, username, password)
		if err != nil {
			http.Error(w, fmt.Sprintf("DVR re-auth failed: %v", err), http.StatusBadGateway)
			return
		}
		dvrURL = fmt.Sprintf("ws://%s/requestWebsocketConnection?sessionID=%s", host, url.QueryEscape(session.ID))
		dvrConn, err = dialDVR(dvrURL, session)
		if err != nil {
			http.Error(w, fmt.Sprintf("DVR WebSocket connect failed: %v", err), http.StatusBadGateway)
			return
		}
	}

	// Upgrade browser connection
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		dvrConn.Close()
		return
	}

	log.Printf("DVR WebSocket proxy established for camera %s (host: %s)", cameraID, host)

	// Bidirectional relay
	done := make(chan struct{})

	// Client → DVR
	go func() {
		defer func() {
			close(done)
			dvrConn.Close()
		}()
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				log.Printf("DVR proxy client→dvr read error: %v", err)
				return
			}
			if msgType == websocket.TextMessage {
				log.Printf("DVR proxy client→dvr text (%d bytes): %.200s", len(msg), msg)
			} else {
				log.Printf("DVR proxy client→dvr binary (%d bytes)", len(msg))
			}
			if err := dvrConn.WriteMessage(msgType, msg); err != nil {
				log.Printf("DVR proxy client→dvr write error: %v", err)
				return
			}
		}
	}()

	// DVR → Client
	var binaryCount int
	go func() {
		defer clientConn.Close()
		for {
			msgType, msg, err := dvrConn.ReadMessage()
			if err != nil {
				log.Printf("DVR proxy dvr→client read error: %v", err)
				return
			}
			if msgType == websocket.TextMessage {
				log.Printf("DVR proxy dvr→client text (%d bytes): %.200s", len(msg), msg)
			} else {
				binaryCount++
				if binaryCount <= 3 || binaryCount%100 == 0 {
					log.Printf("DVR proxy dvr→client binary #%d (%d bytes)", binaryCount, len(msg))
				}
			}
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				log.Printf("DVR proxy dvr→client write error: %v", err)
				return
			}
		}
	}()

	<-done
	log.Printf("DVR WebSocket proxy closed for camera %s", cameraID)
}

func dialDVR(dvrURL string, session *Session) (*websocket.Conn, error) {
	header := http.Header{}
	if len(session.Cookies) > 0 {
		var parts []string
		for _, c := range session.Cookies {
			parts = append(parts, c.Name+"="+c.Value)
		}
		header.Set("Cookie", strings.Join(parts, "; "))
	}
	conn, _, err := websocket.DefaultDialer.Dial(dvrURL, header)
	return conn, err
}

func extractHost(rtspURL string) string {
	u, err := url.Parse(rtspURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

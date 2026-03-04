package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"surveillance-client/internal/config"
)

type Server struct {
	cfg    *config.Config
	mux    *http.ServeMux
	webFS  fs.FS
	deps   *Dependencies
}

type Dependencies struct {
	CameraHandler   http.Handler
	GroupHandler    http.Handler
	DiscoveryHandler http.Handler
	DvrProxyHandler http.Handler
	ExportHandler   http.Handler
}

func New(cfg *config.Config, webAssets embed.FS, deps *Dependencies) *Server {
	webFS, err := fs.Sub(webAssets, "web/dist")
	if err != nil {
		log.Fatal("failed to get web subtree:", err)
	}

	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		webFS: webFS,
		deps:  deps,
	}

	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.cfg.AuthPassword != "" {
		_, pass, ok := r.BasicAuth()
		if !ok || !checkPassword(pass, s.cfg.AuthPassword) {
			w.Header().Set("WWW-Authenticate", `Basic realm="surveillance"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	s.mux.ServeHTTP(w, r)
}

func checkPassword(given, expected string) bool {
	a := sha256.Sum256([]byte(given))
	b := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

func (s *Server) go2rtcProxy() http.Handler {
	target, err := url.Parse(s.cfg.Go2RTCAPI)
	if err != nil {
		log.Fatal("invalid go2rtc API URL:", err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.URL.Path = strings.TrimPrefix(r.In.URL.Path, "/go2rtc")
			r.Out.URL.RawQuery = r.In.URL.RawQuery
			r.Out.Host = target.Host
		},
	}
	return proxy
}

func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.webFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// Try to open the file — if it exists, serve it
		if f, err := s.webFS.Open(path); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

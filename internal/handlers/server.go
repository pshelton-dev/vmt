// Package handlers wires the HTTP layer: the JSON API, the embedded SPA and
// the file-producing endpoints (export, backup, uploads).
package handlers

import (
	"database/sql"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"vmt/internal/auth"
	"vmt/internal/config"
	"vmt/web"
)

type Server struct {
	db    *sql.DB
	auth  *auth.Manager
	mu    sync.RWMutex // guards cfg (display prefs are editable at runtime)
	cfg   config.Config
	files http.Handler
}

// New builds a Server and prepares the static file handler.
func New(db *sql.DB, am *auth.Manager, cfg config.Config) (*Server, error) {
	s := &Server{db: db, auth: am, cfg: cfg}
	s.loadPrefs() // apply any saved display-preference overrides
	static, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		return nil, err
	}
	s.files = http.StripPrefix("/static/", http.FileServer(http.FS(static)))
	return s, nil
}

// currentCfg returns a snapshot of the (mutable) config under a read lock.
func (s *Server) currentCfg() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Routes returns the application's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Static assets shared with the SPA (favicon, vehicle-data.json).
	mux.Handle("GET /static/", s.files)

	// JSON API.
	s.mountAPI(mux)

	// Old /app URLs (pre-cutover bookmarks, installed PWAs) land on the root SPA.
	mux.Handle("/app/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, strings.TrimPrefix(r.URL.Path, "/app"), http.StatusMovedPermanently)
	}))
	mux.Handle("GET /app", http.RedirectHandler("/", http.StatusMovedPermanently))

	// Everything else is the SPA (with client-side routing fallback).
	mux.Handle("/", spaHandler())

	return logRequests(mux)
}

// ---- small helpers ----

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/static/") && !strings.HasPrefix(r.URL.Path, "/assets/") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}

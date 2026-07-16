// Package handlers wires the HTTP layer: routing, rendering and request logic.
package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
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
	tmpl  map[string]*template.Template
	files http.Handler
}

// New builds a Server, parsing templates and preparing the static file handler.
func New(db *sql.DB, am *auth.Manager, cfg config.Config) (*Server, error) {
	s := &Server{db: db, auth: am, cfg: cfg}
	if err := s.parseTemplates(); err != nil {
		return nil, err
	}
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

func (s *Server) parseTemplates() error {
	pages := []string{
		"login", "setup", "error", "dashboard", "vehicles",
		"vehicle_form", "vehicle_detail", "service_form",
		"reminder_form", "reminders", "reports", "settings", "import_preview",
		"reference_form",
	}
	s.tmpl = make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		t, err := template.New("base.html").Funcs(s.funcMap()).ParseFS(
			web.TemplatesFS, "templates/base.html", "templates/"+p+".html",
		)
		if err != nil {
			return fmt.Errorf("parse template %s: %w", p, err)
		}
		s.tmpl[p] = t
	}
	return nil
}

// Routes returns the application's HTTP handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public.
	mux.Handle("GET /static/", s.files)
	mux.HandleFunc("GET /login", s.getLogin)
	mux.HandleFunc("POST /login", s.postLogin)
	mux.HandleFunc("GET /setup", s.getSetup)
	mux.HandleFunc("POST /setup", s.postSetup)
	mux.HandleFunc("POST /logout", s.postLogout)

	// Protected.
	auth := s.requireAuth
	mux.HandleFunc("GET /{$}", auth(s.dashboard))
	mux.HandleFunc("GET /vehicles", auth(s.listVehiclesHandler))
	mux.HandleFunc("GET /vehicles/new", auth(s.newVehicle))
	mux.HandleFunc("POST /vehicles", auth(s.createVehicle))
	mux.HandleFunc("GET /vehicles/{id}", auth(s.showVehicle))
	mux.HandleFunc("GET /vehicles/{id}/edit", auth(s.editVehicle))
	mux.HandleFunc("POST /vehicles/{id}", auth(s.updateVehicle))
	mux.HandleFunc("POST /vehicles/{id}/delete", auth(s.deleteVehicle))
	mux.HandleFunc("POST /vehicles/{id}/photos", auth(s.uploadPhoto))
	mux.HandleFunc("POST /vehicles/{id}/documents", auth(s.uploadDocument))
	mux.HandleFunc("POST /vehicles/{id}/photo/{aid}/primary", auth(s.setPrimaryPhoto))

	mux.HandleFunc("GET /vehicles/{id}/services/new", auth(s.newService))
	mux.HandleFunc("POST /vehicles/{id}/services", auth(s.createService))
	mux.HandleFunc("GET /services/{id}/edit", auth(s.editService))
	mux.HandleFunc("POST /services/{id}", auth(s.updateService))
	mux.HandleFunc("POST /services/{id}/delete", auth(s.deleteService))

	mux.HandleFunc("POST /vehicles/{id}/reference", auth(s.createReference))
	mux.HandleFunc("GET /reference/{id}/edit", auth(s.editReference))
	mux.HandleFunc("POST /reference/{id}", auth(s.updateReference))
	mux.HandleFunc("POST /reference/{id}/delete", auth(s.deleteReference))

	mux.HandleFunc("GET /vehicles/{id}/reminders/new", auth(s.newReminder))
	mux.HandleFunc("POST /vehicles/{id}/reminders", auth(s.createReminder))
	mux.HandleFunc("GET /reminders", auth(s.listReminders))
	mux.HandleFunc("GET /reminders/{id}/edit", auth(s.editReminder))
	mux.HandleFunc("POST /reminders/{id}", auth(s.updateReminder))
	mux.HandleFunc("POST /reminders/{id}/complete", auth(s.completeReminder))
	mux.HandleFunc("POST /reminders/{id}/delete", auth(s.deleteReminder))

	mux.HandleFunc("GET /reports", auth(s.reports))
	mux.HandleFunc("GET /settings", auth(s.settings))
	mux.HandleFunc("POST /settings/password", auth(s.changePassword))
	mux.HandleFunc("POST /settings/preferences", auth(s.savePreferences))
	mux.HandleFunc("POST /settings/notifications", auth(s.saveNotifications))
	mux.HandleFunc("POST /settings/test-email", auth(s.testEmail))
	mux.HandleFunc("POST /settings/import/preview", auth(s.previewImport))
	mux.HandleFunc("POST /settings/import", auth(s.importData))
	mux.HandleFunc("GET /export/services.csv", auth(s.exportAllServices))
	mux.HandleFunc("GET /vehicles/{id}/export.csv", auth(s.exportVehicleServices))
	mux.HandleFunc("GET /settings/backup", auth(s.backup))
	mux.HandleFunc("POST /settings/restore", auth(s.restore))
	mux.HandleFunc("GET /files/{id}", auth(s.serveFile))
	mux.HandleFunc("POST /attachments/{id}/delete", auth(s.deleteAttachment))

	// JSON API (v2 SPA); coexists with the HTML routes until cutover.
	s.mountAPI(mux)

	return logRequests(mux)
}

// ---- rendering ----

type View struct {
	Title  string
	Active string
	Cfg    config.Config
	Flash  string
	Authed bool
	Data   any
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, page string, v View) {
	t, ok := s.tmpl[page]
	if !ok {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	v.Cfg = s.currentCfg()
	v.Authed = s.isAuthed(r)
	if v.Flash == "" {
		v.Flash = s.takeFlash(w, r)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", v); err != nil {
		log.Printf("render %s: %v", page, err)
	}
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	w.WriteHeader(code)
	s.render(w, r, "error", View{
		Title: http.StatusText(code),
		Data:  map[string]any{"Code": code, "Message": msg},
	})
}

// ---- flash messages (one-shot cookie) ----

func (s *Server) setFlash(w http.ResponseWriter, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name: "vmt_flash", Value: url.QueryEscape(msg), Path: "/", MaxAge: 30, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) takeFlash(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie("vmt_flash")
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{Name: "vmt_flash", Value: "", Path: "/", MaxAge: -1})
	v, _ := url.QueryUnescape(c.Value)
	return v
}

// ---- small helpers ----

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func redirect(w http.ResponseWriter, r *http.Request, to string) {
	http.Redirect(w, r, to, http.StatusSeeOther)
}

func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}

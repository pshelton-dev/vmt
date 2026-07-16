package handlers

// The /api/v1 JSON API. It reuses the same store methods and session auth as
// the HTML handlers; the two surfaces coexist until the SPA replaces the
// server-rendered pages (v2 cutover), at which point the HTML handlers go away.

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"vmt/internal/auth"
	"vmt/internal/models"
)

// maxJSONBody caps JSON request bodies; uploads use maxUpload instead.
const maxJSONBody = 1 << 20 // 1 MiB

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("api: encode response: %v", err)
		}
	}
}

func apiError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// decodeJSON strictly decodes a JSON request body into v.
func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxJSONBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errors.New("invalid JSON body: " + err.Error())
	}
	return nil
}

// apiAuth wraps an API handler, rejecting unauthenticated requests with a 401
// JSON body (the SPA redirects to its login screen on 401) rather than the
// HTML handlers' 303-to-/login.
func (s *Server) apiAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		configured, err := s.auth.IsConfigured()
		if err != nil {
			apiError(w, http.StatusInternalServerError, "database error")
			return
		}
		if !configured {
			apiError(w, http.StatusUnauthorized, "setup required")
			return
		}
		if !s.isAuthed(r) {
			apiError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next(w, r)
	}
}

// mountAPI registers all /api/v1 routes on mux.
func (s *Server) mountAPI(mux *http.ServeMux) {
	// Public.
	mux.HandleFunc("GET /api/v1/session", s.apiSession)
	mux.HandleFunc("POST /api/v1/login", s.apiLogin)
	mux.HandleFunc("POST /api/v1/setup", s.apiSetup)
	mux.HandleFunc("POST /api/v1/logout", s.apiLogout)

	a := s.apiAuth
	mux.HandleFunc("GET /api/v1/meta", a(s.apiMeta))
	mux.HandleFunc("GET /api/v1/dashboard", a(s.apiDashboard))

	mux.HandleFunc("GET /api/v1/vehicles", a(s.apiListVehicles))
	mux.HandleFunc("POST /api/v1/vehicles", a(s.apiCreateVehicle))
	mux.HandleFunc("GET /api/v1/vehicles/{id}", a(s.apiGetVehicle))
	mux.HandleFunc("PUT /api/v1/vehicles/{id}", a(s.apiUpdateVehicle))
	mux.HandleFunc("DELETE /api/v1/vehicles/{id}", a(s.apiDeleteVehicle))
	mux.HandleFunc("POST /api/v1/vehicles/{id}/photos", a(s.apiUploadPhoto))
	mux.HandleFunc("POST /api/v1/vehicles/{id}/documents", a(s.apiUploadDocument))
	mux.HandleFunc("POST /api/v1/vehicles/{id}/photo/{aid}/primary", a(s.apiSetPrimaryPhoto))
	mux.HandleFunc("DELETE /api/v1/attachments/{id}", a(s.apiDeleteAttachment))

	mux.HandleFunc("GET /api/v1/vehicles/{id}/services", a(s.apiListServices))
	mux.HandleFunc("POST /api/v1/vehicles/{id}/services", a(s.apiCreateService))
	mux.HandleFunc("PUT /api/v1/services/{id}", a(s.apiUpdateService))
	mux.HandleFunc("DELETE /api/v1/services/{id}", a(s.apiDeleteService))
	mux.HandleFunc("POST /api/v1/services/{id}/attachments", a(s.apiUploadServiceAttachment))

	mux.HandleFunc("GET /api/v1/reminders", a(s.apiListReminders))
	mux.HandleFunc("POST /api/v1/vehicles/{id}/reminders", a(s.apiCreateReminder))
	mux.HandleFunc("PUT /api/v1/reminders/{id}", a(s.apiUpdateReminder))
	mux.HandleFunc("POST /api/v1/reminders/{id}/complete", a(s.apiCompleteReminder))
	mux.HandleFunc("DELETE /api/v1/reminders/{id}", a(s.apiDeleteReminder))

	mux.HandleFunc("POST /api/v1/vehicles/{id}/reference", a(s.apiCreateReference))
	mux.HandleFunc("PUT /api/v1/reference/{id}", a(s.apiUpdateReference))
	mux.HandleFunc("DELETE /api/v1/reference/{id}", a(s.apiDeleteReference))

	mux.HandleFunc("GET /api/v1/reports", a(s.apiReports))

	mux.HandleFunc("GET /api/v1/settings", a(s.apiGetSettings))
	mux.HandleFunc("PUT /api/v1/settings", a(s.apiUpdateSettings))
	mux.HandleFunc("POST /api/v1/settings/password", a(s.apiChangePassword))
	mux.HandleFunc("POST /api/v1/settings/test-email", a(s.apiTestEmail))
	mux.HandleFunc("POST /api/v1/import/preview", a(s.apiImportPreview))
	mux.HandleFunc("POST /api/v1/import", a(s.apiImportCommit))

	// File-producing endpoints reuse the existing streaming handlers; they are
	// not HTML and survive the v2 cutover.
	mux.HandleFunc("GET /api/v1/export/services.csv", a(s.exportAllServices))
	mux.HandleFunc("GET /api/v1/vehicles/{id}/export.csv", a(s.exportVehicleServices))
	mux.HandleFunc("GET /api/v1/backup", a(s.backup))
	mux.HandleFunc("POST /api/v1/restore", a(s.restore))
	mux.HandleFunc("GET /api/v1/files/{id}", a(s.serveFile))
}

// ---- auth ----

func (s *Server) apiSession(w http.ResponseWriter, r *http.Request) {
	configured, err := s.auth.IsConfigured()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{
		"configured": configured,
		"authed":     configured && s.isAuthed(r),
	})
}

func (s *Server) apiLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	ok, err := s.auth.Check(req.Password)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "login failed")
		return
	}
	if !ok {
		apiError(w, http.StatusUnauthorized, "incorrect password")
		return
	}
	if err := s.startSession(w, r); err != nil {
		apiError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiSetup(w http.ResponseWriter, r *http.Request) {
	if configured, _ := s.auth.IsConfigured(); configured {
		apiError(w, http.StatusConflict, "already configured")
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Password) < 6 {
		apiError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}
	if err := s.auth.SetPassword(req.Password); err != nil {
		apiError(w, http.StatusInternalServerError, "could not save password")
		return
	}
	if err := s.startSession(w, r); err != nil {
		apiError(w, http.StatusInternalServerError, "could not start session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName()); err == nil {
		s.auth.Destroy(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.CookieName(), Value: "", Path: "/", MaxAge: -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

// ---- meta & dashboard ----

// apiMeta returns static-ish lookup data the SPA needs to render forms.
func (s *Server) apiMeta(w http.ResponseWriter, r *http.Request) {
	cfg := s.currentCfg()
	presets := make([]map[string]string, 0, len(dateFormatPresets))
	for _, p := range dateFormatPresets {
		presets = append(presets, map[string]string{"layout": p.Layout, "example": p.Example})
	}
	kinds := make([]map[string]string, 0, len(referenceKinds))
	for _, k := range referenceKinds {
		kinds = append(kinds, map[string]string{"value": k.Value, "label": k.Label})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service_categories": models.ServiceCategories,
		"reference_kinds":    kinds,
		"date_presets":       presets,
		"currency":           cfg.Currency,
		"distance_unit":      cfg.DistanceUnit,
		"date_format":        cfg.DateFormat,
	})
}

func (s *Server) apiDashboard(w http.ResponseWriter, r *http.Request) {
	vehicles, err := s.listVehicles()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load vehicles")
		return
	}
	reminders, err := s.listAllReminders()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load reminders")
		return
	}
	due := dueReminders(reminders)
	dueByVehicle := map[int64]int{}
	for _, rem := range due {
		dueByVehicle[rem.VehicleID]++
	}
	var totalCost float64
	var serviceCount int
	for i := range vehicles {
		vehicles[i].DueReminders = dueByVehicle[vehicles[i].ID]
		totalCost += vehicles[i].TotalCost
		serviceCount += vehicles[i].ServiceCount
	}
	recent, _ := s.recentServices(8)
	writeJSON(w, http.StatusOK, map[string]any{
		"vehicles":        emptyIfNil(vehicles),
		"due_reminders":   emptyIfNil(due),
		"recent_services": emptyIfNil(recent),
		"total_cost":      totalCost,
		"service_count":   serviceCount,
	})
}

// emptyIfNil turns a nil slice into an empty one so JSON shows [] not null.
func emptyIfNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

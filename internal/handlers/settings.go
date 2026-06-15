package handlers

import "net/http"

// dateFormatPresets are the selectable display layouts (Go reference time).
var dateFormatPresets = []struct{ Layout, Example string }{
	{"Jan 2, 2006", "May 30, 2026"},
	{"2006-01-02", "2026-05-30"},
	{"01/02/2006", "05/30/2026"},
	{"02/01/2006", "30/05/2026"},
	{"2 Jan 2006", "30 May 2026"},
}

// getPref reads a stored preference, returning def when unset.
func (s *Server) getPref(key, def string) string {
	var v string
	if err := s.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v); err != nil || v == "" {
		return def
	}
	return v
}

func (s *Server) setPref(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings(key, value) VALUES(?, ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

// loadPrefs applies any persisted display-preference overrides over the
// env-derived defaults. Called once at startup.
func (s *Server) loadPrefs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Currency = s.getPref("pref_currency", s.cfg.Currency)
	s.cfg.DistanceUnit = s.getPref("pref_distance_unit", s.cfg.DistanceUnit)
	s.cfg.DateFormat = s.getPref("pref_date_format", s.cfg.DateFormat)
}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	cfg := s.currentCfg()
	s.render(w, r, "settings", View{
		Title:  "Settings",
		Active: "settings",
		Data: map[string]any{
			"Currency":       cfg.Currency,
			"DistanceUnit":   cfg.DistanceUnit,
			"DateFormat":     cfg.DateFormat,
			"DatePresets":    dateFormatPresets,
			"SMTPConfigured": cfg.SMTP.Configured(),
			"SMTPFrom":       cfg.SMTP.From,
			"NotifyEnabled":  s.notifyEnabled(),
			"NotifyEmail":    s.notifyEmail(),
		},
	})
}

func (s *Server) saveNotifications(w http.ResponseWriter, r *http.Request) {
	email := strip(r.FormValue("notify_email"))
	enabled := "0"
	if r.FormValue("notify_enabled") != "" {
		enabled = "1"
	}
	_ = s.setPref("notify_email", email)
	_ = s.setPref("notify_enabled", enabled)
	s.setFlash(w, "Notification settings saved.")
	redirect(w, r, "/settings")
}

func (s *Server) testEmail(w http.ResponseWriter, r *http.Request) {
	// Persist any address typed alongside the test button first.
	if email := strip(r.FormValue("notify_email")); email != "" {
		_ = s.setPref("notify_email", email)
	}
	if err := s.sendTestEmail(); err != nil {
		s.setFlash(w, "Test email failed: "+err.Error())
	} else {
		s.setFlash(w, "Test email sent to "+s.notifyEmail()+".")
	}
	redirect(w, r, "/settings")
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	current := r.FormValue("current")
	next := r.FormValue("new")
	confirm := r.FormValue("confirm")

	ok, err := s.auth.Check(current)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not verify password")
		return
	}
	if !ok {
		s.setFlash(w, "Current password is incorrect.")
		redirect(w, r, "/settings")
		return
	}
	if len(next) < 6 || next != confirm {
		s.setFlash(w, "New password must match and be at least 6 characters.")
		redirect(w, r, "/settings")
		return
	}
	if err := s.auth.SetPassword(next); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save password")
		return
	}
	s.setFlash(w, "Password changed.")
	redirect(w, r, "/settings")
}

func (s *Server) savePreferences(w http.ResponseWriter, r *http.Request) {
	currency := strip(r.FormValue("currency"))
	unit := strip(r.FormValue("distance_unit"))
	format := strip(r.FormValue("date_format"))
	if currency == "" {
		currency = "$"
	}
	if unit == "" {
		unit = "mi"
	}
	if !validDateFormat(format) {
		format = "Jan 2, 2006"
	}

	_ = s.setPref("pref_currency", currency)
	_ = s.setPref("pref_distance_unit", unit)
	_ = s.setPref("pref_date_format", format)

	s.mu.Lock()
	s.cfg.Currency = currency
	s.cfg.DistanceUnit = unit
	s.cfg.DateFormat = format
	s.mu.Unlock()

	s.setFlash(w, "Preferences saved.")
	redirect(w, r, "/settings")
}

func validDateFormat(f string) bool {
	for _, p := range dateFormatPresets {
		if p.Layout == f {
			return true
		}
	}
	return false
}

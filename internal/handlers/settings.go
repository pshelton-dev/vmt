package handlers

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

func validDateFormat(f string) bool {
	for _, p := range dateFormatPresets {
		if p.Layout == f {
			return true
		}
	}
	return false
}

package handlers

import (
	"io"
	"net/http"
	"strings"
)

func (s *Server) apiGetSettings(w http.ResponseWriter, r *http.Request) {
	cfg := s.currentCfg()
	smtp := s.smtpSettings()
	gmail := s.gmailSettings()
	redirect, redirectErr := s.oauthRedirectURI()

	out := map[string]any{
		"currency":      cfg.Currency,
		"distance_unit": cfg.DistanceUnit,
		"date_format":   cfg.DateFormat,

		// Mail: which provider is active and whether it can actually send.
		"mail_provider":   s.mailProvider(),
		"mail_configured": s.mailConfigured(),
		"mail_from":       s.mailFrom(),

		// SMTP. Secrets are never returned — only whether one is stored.
		"smtp_host":         smtp.Host,
		"smtp_port":         smtp.Port,
		"smtp_user":         smtp.User,
		"smtp_from":         smtp.From,
		"smtp_tls":          smtp.TLS,
		"smtp_insecure":     smtp.Insecure,
		"smtp_pass_set":     smtp.Pass != "",
		"smtp_configured":   smtp.Configured(),

		// Gmail. Same rule: the client secret and refresh token stay server-side.
		"gmail_client_id":         gmail.ClientID,
		"gmail_client_secret_set": gmail.ClientSecret != "",
		"gmail_connected":         gmail.Configured(),
		"gmail_email":             gmail.From,
		"gmail_redirect_uri":      redirect,

		"notify_enabled": s.notifyEnabled(),
		"notify_email":   s.notifyEmail(),
	}
	// Surface the reason the OAuth flow can't start, so the UI can explain it
	// instead of failing at the click.
	if redirectErr != nil {
		out["gmail_redirect_uri"] = ""
		out["gmail_setup_error"] = redirectErr.Error()
	}
	writeJSON(w, http.StatusOK, out)
}

// apiUpdateSettings saves display preferences and/or notification settings.
// Only fields present in the body are changed.
func (s *Server) apiUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Currency      *string `json:"currency"`
		DistanceUnit  *string `json:"distance_unit"`
		DateFormat    *string `json:"date_format"`
		NotifyEnabled *bool   `json:"notify_enabled"`
		NotifyEmail   *string `json:"notify_email"`

		MailProvider *string `json:"mail_provider"`

		SMTPHost     *string `json:"smtp_host"`
		SMTPPort     *string `json:"smtp_port"`
		SMTPUser     *string `json:"smtp_user"`
		SMTPPass     *string `json:"smtp_pass"` // write-only; "" clears
		SMTPFrom     *string `json:"smtp_from"`
		SMTPTLS      *string `json:"smtp_tls"`
		SMTPInsecure *bool   `json:"smtp_insecure"`

		GmailClientID     *string `json:"gmail_client_id"`
		GmailClientSecret *string `json:"gmail_client_secret"` // write-only
	}
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}

	if in.Currency != nil || in.DistanceUnit != nil || in.DateFormat != nil {
		cfg := s.currentCfg()
		currency, unit, format := cfg.Currency, cfg.DistanceUnit, cfg.DateFormat
		if in.Currency != nil {
			currency = strip(*in.Currency)
		}
		if in.DistanceUnit != nil {
			unit = strip(*in.DistanceUnit)
		}
		if in.DateFormat != nil {
			format = strip(*in.DateFormat)
		}
		if currency == "" {
			currency = "$"
		}
		if unit == "" {
			unit = "mi"
		}
		if !validDateFormat(format) {
			apiError(w, http.StatusBadRequest, "unknown date format (see /api/v1/meta date_presets)")
			return
		}
		_ = s.setPref("pref_currency", currency)
		_ = s.setPref("pref_distance_unit", unit)
		_ = s.setPref("pref_date_format", format)
		s.mu.Lock()
		s.cfg.Currency = currency
		s.cfg.DistanceUnit = unit
		s.cfg.DateFormat = format
		s.mu.Unlock()
	}

	if in.NotifyEmail != nil {
		_ = s.setPref("notify_email", strip(*in.NotifyEmail))
	}
	if in.NotifyEnabled != nil {
		_ = s.setPref("notify_enabled", boolPref(*in.NotifyEnabled))
	}

	// ---- mail provider ----
	if in.MailProvider != nil {
		p := strings.ToLower(strip(*in.MailProvider))
		if p != providerSMTP && p != providerGmail {
			apiError(w, http.StatusBadRequest, `mail_provider must be "smtp" or "gmail"`)
			return
		}
		if p == providerGmail && !s.gmailSettings().Configured() {
			apiError(w, http.StatusBadRequest, "connect a Google account before switching to Gmail")
			return
		}
		_ = s.setPref(prefMailProvider, p)
	}

	// ---- SMTP ----
	if in.SMTPTLS != nil {
		mode := strings.ToLower(strip(*in.SMTPTLS))
		switch mode {
		case "starttls", "implicit", "none":
			_ = s.setPref(prefSMTPTLS, mode)
		default:
			apiError(w, http.StatusBadRequest, `smtp_tls must be "starttls", "implicit" or "none"`)
			return
		}
	}
	setIf(s, prefSMTPHost, in.SMTPHost)
	setIf(s, prefSMTPPort, in.SMTPPort)
	setIf(s, prefSMTPUser, in.SMTPUser)
	setIf(s, prefSMTPFrom, in.SMTPFrom)
	if in.SMTPPass != nil {
		// Sent only when the user types a new one; empty string clears it.
		_ = s.setPref(prefSMTPPass, *in.SMTPPass)
	}
	if in.SMTPInsecure != nil {
		_ = s.setPref(prefSMTPInsecure, boolPref(*in.SMTPInsecure))
	}

	// ---- Gmail ----
	setIf(s, prefGmailClientID, in.GmailClientID)
	if in.GmailClientSecret != nil {
		_ = s.setPref(prefGmailClientSecret, *in.GmailClientSecret)
	}

	s.apiGetSettings(w, r)
}

// setIf stores a trimmed preference when the field was present in the request.
func setIf(s *Server, key string, v *string) {
	if v != nil {
		_ = s.setPref(key, strip(*v))
	}
}

func boolPref(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (s *Server) apiChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	ok, err := s.auth.Check(in.Current)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not verify password")
		return
	}
	if !ok {
		apiError(w, http.StatusForbidden, "current password is incorrect")
		return
	}
	if len(in.New) < 6 {
		apiError(w, http.StatusBadRequest, "new password must be at least 6 characters")
		return
	}
	if err := s.auth.SetPassword(in.New); err != nil {
		apiError(w, http.StatusInternalServerError, "could not save password")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiTestEmail(w http.ResponseWriter, r *http.Request) {
	if err := s.sendTestEmail(); err != nil {
		apiError(w, http.StatusBadGateway, "test email failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"sent_to": s.notifyEmail()})
}

// ---- CSV import ----

// readImportCSV extracts CSV content from either a multipart "csv" file field
// or a JSON body {"csv_data": "..."} (used by the SPA's confirm step).
func readImportCSV(r *http.Request) (io.Reader, bool) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var in struct {
			CSVData string `json:"csv_data"`
		}
		if err := decodeJSON(r, &in); err != nil || in.CSVData == "" {
			return nil, false
		}
		return strings.NewReader(in.CSVData), true
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		return nil, false
	}
	file, _, err := r.FormFile("csv")
	if err != nil {
		return nil, false
	}
	return io.LimitReader(file, maxUpload), true
}

// apiImportPreview parses the CSV and reports what an import would do,
// without writing anything. The response echoes the raw CSV so the client can
// send it back verbatim on confirm.
func (s *Server) apiImportPreview(w http.ResponseWriter, r *http.Request) {
	reader, ok := readImportCSV(r)
	if !ok {
		apiError(w, http.StatusBadRequest, "provide a CSV file (multipart \"csv\") or JSON {\"csv_data\": ...}")
		return
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not read file")
		return
	}
	rows, err := parseServiceRows(strings.NewReader(string(raw)))
	if err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	annotated, newVehicles, wouldImport, skipped := s.planImport(rows)

	type previewRow struct {
		Line        int     `json:"line"`
		OK          bool    `json:"ok"`
		Reason      string  `json:"reason,omitempty"`
		Vehicle     string  `json:"vehicle,omitempty"`
		NewVehicle  bool    `json:"new_vehicle,omitempty"`
		Date        string  `json:"date,omitempty"`
		Odometer    *int64  `json:"odometer,omitempty"`
		Category    string  `json:"category,omitempty"`
		Description string  `json:"description,omitempty"`
		Vendor      string  `json:"vendor,omitempty"`
		Cost        float64 `json:"cost,omitempty"`
		Notes       string  `json:"notes,omitempty"`
	}
	out := make([]previewRow, len(annotated))
	for i, row := range annotated {
		out[i] = previewRow{
			Line: row.Line, OK: row.OK, Reason: row.Reason,
			Vehicle: row.Vehicle, NewVehicle: row.NewVehicle,
			Date: row.Date, Odometer: row.Odometer, Category: row.Category,
			Description: row.Description, Vendor: row.Vendor,
			Cost: row.Cost, Notes: row.Notes,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rows":         out,
		"new_vehicles": emptyIfNil(newVehicles),
		"would_import": wouldImport,
		"skipped":      skipped,
		"csv_data":     string(raw),
	})
}

// apiImportCommit performs the import and reports what happened.
func (s *Server) apiImportCommit(w http.ResponseWriter, r *http.Request) {
	reader, ok := readImportCSV(r)
	if !ok {
		apiError(w, http.StatusBadRequest, "provide a CSV file (multipart \"csv\") or JSON {\"csv_data\": ...}")
		return
	}
	rows, err := parseServiceRows(reader)
	if err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	res := s.commitImport(rows)
	writeJSON(w, http.StatusOK, map[string]any{
		"imported":         res.Imported,
		"vehicles_created": res.VehiclesCreated,
		"skipped":          res.Skipped,
		"issues":           emptyIfNil(res.Issues),
		"summary":          res.summary(),
	})
}

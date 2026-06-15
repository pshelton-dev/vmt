// Package config loads runtime configuration from the environment.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Addr          string        // listen address, e.g. ":8080"
	DataDir       string        // base data directory (db + uploads)
	DBPath        string        // sqlite file path
	UploadDir     string        // where uploaded files live
	AdminPassword string        // optional bootstrap password from env
	DistanceUnit  string        // display unit for odometer, e.g. "mi"
	Currency      string        // currency symbol, e.g. "$"
	DateFormat    string        // Go time layout for display
	SessionTTL    time.Duration // how long a login lasts
	BaseURL       string        // public base URL for links in emails (optional)
	SMTP          SMTP          // outbound email settings
}

// SMTP holds outbound mail server settings (all from environment variables).
type SMTP struct {
	Host string
	Port string
	User string
	Pass string
	From string
	TLS  string // "starttls" (default), "implicit", or "none"
}

// Configured reports whether enough SMTP settings are present to send mail.
func (s SMTP) Configured() bool {
	return s.Host != "" && s.From != ""
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load builds a Config from environment variables, applying sensible defaults.
func Load() Config {
	dataDir := env("VMT_DATA_DIR", "./data")
	return Config{
		Addr:          env("VMT_ADDR", ":8080"),
		DataDir:       dataDir,
		DBPath:        filepath.Join(dataDir, "vmt.db"),
		UploadDir:     filepath.Join(dataDir, "uploads"),
		AdminPassword: os.Getenv("VMT_ADMIN_PASSWORD"),
		DistanceUnit:  env("VMT_DISTANCE_UNIT", "mi"),
		Currency:      env("VMT_CURRENCY", "$"),
		DateFormat:    env("VMT_DATE_FORMAT", "Jan 2, 2006"),
		SessionTTL:    30 * 24 * time.Hour,
		BaseURL:       strings.TrimRight(os.Getenv("VMT_BASE_URL"), "/"),
		SMTP: SMTP{
			Host: os.Getenv("VMT_SMTP_HOST"),
			Port: env("VMT_SMTP_PORT", "587"),
			User: os.Getenv("VMT_SMTP_USER"),
			Pass: os.Getenv("VMT_SMTP_PASS"),
			From: os.Getenv("VMT_SMTP_FROM"),
			TLS:  env("VMT_SMTP_TLS", "starttls"),
		},
	}
}

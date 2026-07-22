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
	}
}

// Command vmt is a self-hosted vehicle maintenance tracker.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vmt/internal/auth"
	"vmt/internal/config"
	"vmt/internal/db"
	"vmt/internal/handlers"
)

func main() {
	log.SetFlags(log.LstdFlags)
	cfg := config.Load()

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	am := auth.New(database, cfg.SessionTTL)
	if set, err := am.EnsurePassword(cfg.AdminPassword); err != nil {
		log.Fatalf("set password: %v", err)
	} else if set {
		log.Print("admin password initialised from VMT_ADMIN_PASSWORD")
	}

	srv, err := handlers.New(database, am, cfg)
	if err != nil {
		log.Fatalf("init server: %v", err)
	}

	// Background loop that emails due reminders (no-op until configured).
	notifyCtx, stopNotify := context.WithCancel(context.Background())
	defer stopNotify()
	srv.StartNotifier(notifyCtx)
	if cfg.SMTP.Configured() {
		log.Print("SMTP configured; reminder email notifications available")
	}

	// Periodically purge expired sessions.
	go func() {
		t := time.NewTicker(6 * time.Hour)
		defer t.Stop()
		for range t.C {
			am.CleanupExpired()
		}
	}()

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		// Generous write timeout to allow large file downloads/uploads.
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}

	go func() {
		log.Printf("VMT listening on %s (data dir: %s)", cfg.Addr, cfg.DataDir)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Print("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

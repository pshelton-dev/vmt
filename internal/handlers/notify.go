package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"vmt/internal/mail"
	"vmt/internal/models"
)

// reNotifyAfter is how long before a still-due reminder is emailed again.
const reNotifyAfter = 7 * 24 * time.Hour

// notifyEnabled reports the user's stored on/off toggle for email notifications.
func (s *Server) notifyEnabled() bool {
	return s.getPref("notify_enabled", "0") == "1"
}

func (s *Server) notifyEmail() string {
	return s.getPref("notify_email", "")
}

// mailReady reports whether a due-reminder email could actually be sent:
// SMTP configured (env), notifications enabled, and a recipient set.
func (s *Server) mailReady() bool {
	return s.cfg.SMTP.Configured() && s.notifyEnabled() && s.notifyEmail() != ""
}

// StartNotifier launches the background loop that emails due reminders. It
// returns immediately; the loop stops when ctx is cancelled.
func (s *Server) StartNotifier(ctx context.Context) {
	go func() {
		// Small initial delay so startup logs settle; then check periodically.
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				s.runNotifications()
				timer.Reset(6 * time.Hour)
			}
		}
	}()
}

// runNotifications sends one email per due/overdue reminder that has opted in
// and hasn't been notified recently.
func (s *Server) runNotifications() {
	if !s.mailReady() {
		return
	}
	to := s.notifyEmail()
	reminders, err := s.listAllReminders()
	if err != nil {
		log.Printf("notify: load reminders: %v", err)
		return
	}
	today := time.Now().Format("2006-01-02")
	sent := 0
	for _, r := range reminders {
		if !r.Notify || (r.Status != "due" && r.Status != "overdue") {
			continue
		}
		if recentlyNotified(r.LastNotified) {
			continue
		}
		subject, body := s.reminderEmail(r)
		if err := mail.Send(s.cfg.SMTP, to, subject, body); err != nil {
			log.Printf("notify: send reminder %d: %v", r.ID, err)
			continue
		}
		if err := s.markNotified(r.ID, today); err != nil {
			log.Printf("notify: mark reminder %d: %v", r.ID, err)
		}
		sent++
	}
	if sent > 0 {
		log.Printf("notify: sent %d reminder email(s) to %s", sent, to)
	}
}

func recentlyNotified(last *string) bool {
	if last == nil {
		return false
	}
	t, err := time.Parse("2006-01-02", *last)
	if err != nil {
		return false
	}
	return time.Since(t) < reNotifyAfter
}

func (s *Server) reminderEmail(r models.Reminder) (subject, body string) {
	subject = fmt.Sprintf("VMT reminder: %s — %s (%s)", r.VehicleName, r.Title, r.StatusText)
	b := fmt.Sprintf("Maintenance reminder for %s\n\n", r.VehicleName)
	b += fmt.Sprintf("  %s\n  Status: %s\n", r.Title, r.StatusText)
	if r.DueDate != nil {
		b += fmt.Sprintf("  Due date: %s\n", *r.DueDate)
	}
	if r.DueOdometer != nil {
		b += fmt.Sprintf("  Due at: %s %s\n", commaInt(*r.DueOdometer), s.cfg.DistanceUnit)
	}
	if r.Notes != "" {
		b += fmt.Sprintf("  Notes: %s\n", r.Notes)
	}
	if s.cfg.BaseURL != "" {
		b += fmt.Sprintf("\nView vehicle: %s/vehicles/%d\n", s.cfg.BaseURL, r.VehicleID)
	}
	b += "\n— VMT (Vehicle Maintenance Tracker)\n"
	return subject, b
}

// sendTestEmail delivers a test message to the configured recipient.
func (s *Server) sendTestEmail() error {
	to := s.notifyEmail()
	if to == "" {
		return fmt.Errorf("no recipient address set")
	}
	if !s.cfg.SMTP.Configured() {
		return fmt.Errorf("SMTP is not configured (set VMT_SMTP_* env vars)")
	}
	return mail.Send(s.cfg.SMTP, to,
		"VMT test email",
		"This is a test email from your Vehicle Maintenance Tracker.\n\n"+
			"If you received this, reminder notifications are working.\n\n— VMT\n")
}

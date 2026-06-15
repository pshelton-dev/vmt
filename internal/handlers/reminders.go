package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"vmt/internal/models"
)

// statusRank orders reminders by urgency for sorting (lower = more urgent).
var statusRank = map[string]int{"overdue": 0, "due": 1, "soon": 2, "ok": 3}

// annotateReminder fills Status/StatusText based on due date, due odometer and
// the vehicle's current odometer. The most urgent of the date/mileage triggers
// wins.
func annotateReminder(r *models.Reminder, odometer int64) {
	r.Status = "ok"
	r.StatusText = "Scheduled"
	best := 99

	apply := func(status, text string) {
		if statusRank[status] < best {
			best = statusRank[status]
			r.Status = status
			r.StatusText = text
		}
	}

	if r.DueDate != nil {
		if due, err := time.Parse("2006-01-02", *r.DueDate); err == nil {
			today := time.Now().Truncate(24 * time.Hour)
			due = due.Truncate(24 * time.Hour)
			days := int(due.Sub(today).Hours() / 24)
			switch {
			case days < 0:
				apply("overdue", fmt.Sprintf("Overdue %dd", -days))
			case days == 0:
				apply("due", "Due today")
			case days <= 7:
				apply("due", fmt.Sprintf("Due in %dd", days))
			case days <= 30:
				apply("soon", fmt.Sprintf("In %dd", days))
			default:
				apply("ok", "Due "+due.Format("Jan 2"))
			}
		}
	}

	if r.DueOdometer != nil {
		rem := *r.DueOdometer - odometer
		switch {
		case rem < 0:
			apply("overdue", fmt.Sprintf("Overdue %s mi", commaInt(-rem)))
		case rem <= 200:
			apply("due", fmt.Sprintf("In %s mi", commaInt(rem)))
		case rem <= 1000:
			apply("soon", fmt.Sprintf("In %s mi", commaInt(rem)))
		default:
			apply("ok", fmt.Sprintf("In %s mi", commaInt(rem)))
		}
	}
}

// dueReminders filters and sorts reminders that need attention (soon or worse).
func dueReminders(rs []models.Reminder) []models.Reminder {
	var out []models.Reminder
	for _, r := range rs {
		if r.Status != "ok" {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return statusRank[out[i].Status] < statusRank[out[j].Status]
	})
	return out
}

func (s *Server) listReminders(w http.ResponseWriter, r *http.Request) {
	rems, err := s.listAllReminders()
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not load reminders")
		return
	}
	sort.SliceStable(rems, func(i, j int) bool {
		return statusRank[rems[i].Status] < statusRank[rems[j].Status]
	})
	s.render(w, r, "reminders", View{Title: "Reminders", Active: "reminders", Data: rems})
}

func (s *Server) newReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad vehicle id")
		return
	}
	v, err := s.getVehicle(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "vehicle not found")
		return
	}
	s.render(w, r, "reminder_form", View{
		Title:  "Add reminder",
		Active: "vehicles",
		Data: map[string]any{
			"Reminder":  models.Reminder{VehicleID: id},
			"Vehicle":   v,
			"IsNew":     true,
			"Action":    fmt.Sprintf("/vehicles/%d/reminders", id),
			"MailReady": s.mailReady(),
		},
	})
}

func (s *Server) createReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad vehicle id")
		return
	}
	rem := s.parseReminderForm(r)
	rem.VehicleID = id
	if rem.Title == "" {
		s.setFlash(w, "Title is required.")
		redirect(w, r, fmt.Sprintf("/vehicles/%d/reminders/new", id))
		return
	}
	if _, err := s.insertReminder(rem); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save reminder")
		return
	}
	s.setFlash(w, "Reminder added.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
}

func (s *Server) editReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	rem, err := s.getReminder(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "reminder not found")
		return
	}
	v, err := s.getVehicle(rem.VehicleID)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "vehicle not found")
		return
	}
	s.render(w, r, "reminder_form", View{
		Title:  "Edit reminder",
		Active: "vehicles",
		Data: map[string]any{
			"Reminder":  rem,
			"Vehicle":   v,
			"IsNew":     false,
			"Action":    fmt.Sprintf("/reminders/%d", id),
			"MailReady": s.mailReady(),
		},
	})
}

func (s *Server) updateReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	existing, err := s.getReminder(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "reminder not found")
		return
	}
	rem := s.parseReminderForm(r)
	rem.ID = id
	rem.VehicleID = existing.VehicleID
	if rem.Title == "" {
		s.setFlash(w, "Title is required.")
		redirect(w, r, fmt.Sprintf("/reminders/%d/edit", id))
		return
	}
	if err := s.updateReminderRow(rem); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save reminder")
		return
	}
	s.setFlash(w, "Reminder updated.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", existing.VehicleID))
}

func (s *Server) completeReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	rem, err := s.getReminder(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "reminder not found")
		return
	}
	_ = s.completeReminderRow(id)
	// If recurring, schedule the next occurrence automatically.
	s.rollRecurring(rem)
	s.setFlash(w, "Marked done.")
	redirect(w, r, refererOr(r, fmt.Sprintf("/vehicles/%d", rem.VehicleID)))
}

func (s *Server) deleteReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	rem, err := s.getReminder(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "reminder not found")
		return
	}
	_ = s.deleteReminderRow(id)
	s.setFlash(w, "Reminder deleted.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", rem.VehicleID))
}

// rollRecurring creates a follow-up reminder when intervals are configured.
func (s *Server) rollRecurring(rem models.Reminder) {
	if rem.IntervalMonths == nil && rem.IntervalMiles == nil {
		return
	}
	next := models.Reminder{
		VehicleID:      rem.VehicleID,
		Title:          rem.Title,
		Notes:          rem.Notes,
		IntervalMonths: rem.IntervalMonths,
		IntervalMiles:  rem.IntervalMiles,
		Notify:         rem.Notify,
	}
	if rem.IntervalMonths != nil {
		d := time.Now().AddDate(0, *rem.IntervalMonths, 0).Format("2006-01-02")
		next.DueDate = &d
	}
	if rem.IntervalMiles != nil {
		if v, err := s.getVehicle(rem.VehicleID); err == nil {
			due := v.Odometer + int64(*rem.IntervalMiles)
			next.DueOdometer = &due
		}
	}
	_, _ = s.insertReminder(next)
}

func (s *Server) parseReminderForm(r *http.Request) models.Reminder {
	rem := models.Reminder{
		Title:  strip(r.FormValue("title")),
		Notes:  strip(r.FormValue("notes")),
		Notify: r.FormValue("notify") != "",
	}
	rem.DueDate = optDate(r.FormValue("due_date"))
	rem.DueOdometer = optInt64(r.FormValue("due_odometer"))
	rem.IntervalMonths = optInt(r.FormValue("interval_months"))
	rem.IntervalMiles = optInt(r.FormValue("interval_miles"))
	return rem
}

func refererOr(r *http.Request, fallback string) string {
	if ref := r.Header.Get("Referer"); ref != "" {
		return ref
	}
	return fallback
}

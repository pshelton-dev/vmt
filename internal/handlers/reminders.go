package handlers

import (
	"fmt"
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

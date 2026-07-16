package handlers

import (
	"net/http"
	"sort"

	"vmt/internal/models"
)

// reminderInput is the JSON create/update body for a reminder.
type reminderInput struct {
	Title          string  `json:"title"`
	DueDate        *string `json:"due_date"`
	DueOdometer    *int64  `json:"due_odometer"`
	IntervalMonths *int    `json:"interval_months"`
	IntervalMiles  *int    `json:"interval_miles"`
	Notes          string  `json:"notes"`
	Notify         bool    `json:"notify"`
}

func (in reminderInput) toModel() models.Reminder {
	rem := models.Reminder{
		Title:          strip(in.Title),
		Notes:          strip(in.Notes),
		Notify:         in.Notify,
		DueOdometer:    in.DueOdometer,
		IntervalMonths: in.IntervalMonths,
		IntervalMiles:  in.IntervalMiles,
	}
	if in.DueDate != nil {
		rem.DueDate = optDate(*in.DueDate)
	}
	return rem
}

func (s *Server) apiListReminders(w http.ResponseWriter, r *http.Request) {
	rems, err := s.listAllReminders()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load reminders")
		return
	}
	sort.SliceStable(rems, func(i, j int) bool {
		return statusRank[rems[i].Status] < statusRank[rems[j].Status]
	})
	writeJSON(w, http.StatusOK, emptyIfNil(rems))
}

func (s *Server) apiCreateReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	v, err := s.getVehicle(id)
	if err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	var in reminderInput
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	rem := in.toModel()
	rem.VehicleID = id
	if rem.Title == "" {
		apiError(w, http.StatusBadRequest, "title is required")
		return
	}
	rid, err := s.insertReminder(rem)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not save reminder")
		return
	}
	created, err := s.getReminder(rid)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load reminder")
		return
	}
	annotateReminder(&created, v.Odometer)
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) apiUpdateReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	existing, err := s.getReminder(id)
	if err != nil {
		apiError(w, http.StatusNotFound, "reminder not found")
		return
	}
	var in reminderInput
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	rem := in.toModel()
	rem.ID = id
	rem.VehicleID = existing.VehicleID
	if rem.Title == "" {
		apiError(w, http.StatusBadRequest, "title is required")
		return
	}
	if err := s.updateReminderRow(rem); err != nil {
		apiError(w, http.StatusInternalServerError, "could not save reminder")
		return
	}
	updated, err := s.getReminder(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load reminder")
		return
	}
	if v, err := s.getVehicle(updated.VehicleID); err == nil {
		annotateReminder(&updated, v.Odometer)
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) apiCompleteReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	rem, err := s.getReminder(id)
	if err != nil {
		apiError(w, http.StatusNotFound, "reminder not found")
		return
	}
	if err := s.completeReminderRow(id); err != nil {
		apiError(w, http.StatusInternalServerError, "could not complete reminder")
		return
	}
	// If recurring, the next occurrence is scheduled automatically.
	s.rollRecurring(rem)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiDeleteReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getReminder(id); err != nil {
		apiError(w, http.StatusNotFound, "reminder not found")
		return
	}
	if err := s.deleteReminderRow(id); err != nil {
		apiError(w, http.StatusInternalServerError, "could not delete reminder")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

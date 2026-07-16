package handlers

import (
	"net/http"

	"vmt/internal/models"
)

// serviceInput is the JSON create/update body for a service record.
type serviceInput struct {
	Date        string  `json:"date"`
	Odometer    *int64  `json:"odometer"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Vendor      string  `json:"vendor"`
	Cost        float64 `json:"cost"`
	Notes       string  `json:"notes"`
}

func (in serviceInput) toModel() models.ServiceRecord {
	cat := strip(in.Category)
	if cat == "" {
		cat = "Other"
	}
	sr := models.ServiceRecord{
		Category:    cat,
		Description: strip(in.Description),
		Vendor:      strip(in.Vendor),
		Cost:        in.Cost,
		Notes:       strip(in.Notes),
		Odometer:    in.Odometer,
	}
	if d := optDate(in.Date); d != nil {
		sr.Date = *d
	}
	return sr
}

func (s *Server) apiListServices(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	services, err := s.listServices(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load services")
		return
	}
	writeJSON(w, http.StatusOK, emptyIfNil(services))
}

func (s *Server) apiCreateService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	var in serviceInput
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	sr := in.toModel()
	sr.VehicleID = id
	if sr.Description == "" || sr.Date == "" {
		apiError(w, http.StatusBadRequest, "date (yyyy-mm-dd) and description are required")
		return
	}
	sid, err := s.insertService(sr)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not save service")
		return
	}
	if sr.Odometer != nil {
		s.bumpOdometer(id, *sr.Odometer)
	}
	created, err := s.getService(sid)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load service")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) apiUpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	existing, err := s.getService(id)
	if err != nil {
		apiError(w, http.StatusNotFound, "service not found")
		return
	}
	var in serviceInput
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	sr := in.toModel()
	sr.ID = id
	sr.VehicleID = existing.VehicleID
	if sr.Description == "" || sr.Date == "" {
		apiError(w, http.StatusBadRequest, "date (yyyy-mm-dd) and description are required")
		return
	}
	if err := s.updateServiceRow(sr); err != nil {
		apiError(w, http.StatusInternalServerError, "could not save service")
		return
	}
	if sr.Odometer != nil {
		s.bumpOdometer(existing.VehicleID, *sr.Odometer)
	}
	updated, err := s.getService(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load service")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) apiDeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getService(id); err != nil {
		apiError(w, http.StatusNotFound, "service not found")
		return
	}
	atts, _ := s.listAttachmentsByService(id)
	if err := s.deleteServiceRow(id); err != nil {
		apiError(w, http.StatusInternalServerError, "could not delete service")
		return
	}
	for _, a := range atts {
		s.removeStored(a.StoredName)
	}
	w.WriteHeader(http.StatusNoContent)
}

package handlers

import (
	"fmt"
	"net/http"
	"time"

	"vmt/internal/models"
)

func (s *Server) newService(w http.ResponseWriter, r *http.Request) {
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
	blank := models.ServiceRecord{
		VehicleID: id,
		Date:      time.Now().Format("2006-01-02"),
		Category:  "Oil Change",
	}
	if v.Odometer > 0 {
		odo := v.Odometer
		blank.Odometer = &odo
	}
	s.render(w, r, "service_form", View{
		Title:  "Add service",
		Active: "vehicles",
		Data: map[string]any{
			"Service":    blank,
			"Vehicle":    v,
			"IsNew":      true,
			"Action":     fmt.Sprintf("/vehicles/%d/services", id),
			"Categories": models.ServiceCategories,
		},
	})
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad vehicle id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		s.renderError(w, r, http.StatusNotFound, "vehicle not found")
		return
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad form")
		return
	}
	sr := parseServiceForm(r)
	sr.VehicleID = id
	if sr.Description == "" || sr.Date == "" {
		s.setFlash(w, "Date and description are required.")
		redirect(w, r, fmt.Sprintf("/vehicles/%d/services/new", id))
		return
	}
	sid, err := s.insertService(sr)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save service")
		return
	}
	if sr.Odometer != nil {
		s.bumpOdometer(id, *sr.Odometer)
	}
	if _, err := s.saveUpload(r, "document", "document", &id, &sid); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save attachment")
		return
	}
	s.setFlash(w, "Service record added.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
}

func (s *Server) editService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	sr, err := s.getService(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "service not found")
		return
	}
	v, err := s.getVehicle(sr.VehicleID)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "vehicle not found")
		return
	}
	s.render(w, r, "service_form", View{
		Title:  "Edit service",
		Active: "vehicles",
		Data: map[string]any{
			"Service":    sr,
			"Vehicle":    v,
			"IsNew":      false,
			"Action":     fmt.Sprintf("/services/%d", id),
			"Categories": models.ServiceCategories,
		},
	})
}

func (s *Server) updateService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	existing, err := s.getService(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "service not found")
		return
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad form")
		return
	}
	sr := parseServiceForm(r)
	sr.ID = id
	sr.VehicleID = existing.VehicleID
	if sr.Description == "" || sr.Date == "" {
		s.setFlash(w, "Date and description are required.")
		redirect(w, r, fmt.Sprintf("/services/%d/edit", id))
		return
	}
	if err := s.updateServiceRow(sr); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save service")
		return
	}
	if sr.Odometer != nil {
		s.bumpOdometer(existing.VehicleID, *sr.Odometer)
	}
	vid := existing.VehicleID
	if _, err := s.saveUpload(r, "document", "document", &vid, &id); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save attachment")
		return
	}
	s.setFlash(w, "Service record updated.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", existing.VehicleID))
}

func (s *Server) deleteService(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	sr, err := s.getService(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "service not found")
		return
	}
	// Remove attachment files tied to this service.
	atts, _ := s.listAttachmentsByService(id)
	if err := s.deleteServiceRow(id); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not delete service")
		return
	}
	for _, a := range atts {
		s.removeStored(a.StoredName)
	}
	s.setFlash(w, "Service record deleted.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", sr.VehicleID))
}

func parseServiceForm(r *http.Request) models.ServiceRecord {
	cat := strip(r.FormValue("category"))
	if cat == "" {
		cat = "Other"
	}
	return models.ServiceRecord{
		Date:        strip(r.FormValue("date")),
		Odometer:    optInt64(r.FormValue("odometer")),
		Category:    cat,
		Description: strip(r.FormValue("description")),
		Vendor:      strip(r.FormValue("vendor")),
		Cost:        parseFloat(r.FormValue("cost")),
		Notes:       strip(r.FormValue("notes")),
	}
}

package handlers

import (
	"fmt"
	"net/http"

	"vmt/internal/models"
)

// referenceKinds are the selectable item types.
var referenceKinds = []struct{ Value, Label string }{
	{"part", "Part / filter"},
	{"fluid", "Fluid"},
}

func parseReferenceForm(r *http.Request) models.ReferenceItem {
	kind := strip(r.FormValue("kind"))
	if kind != "fluid" {
		kind = "part"
	}
	return models.ReferenceItem{
		Kind:         kind,
		Name:         strip(r.FormValue("name")),
		PartNumber:   strip(r.FormValue("part_number")),
		Manufacturer: strip(r.FormValue("manufacturer")),
		Capacity:     strip(r.FormValue("capacity")),
		Spec:         strip(r.FormValue("spec")),
		Notes:        strip(r.FormValue("notes")),
	}
}

func (s *Server) createReference(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad vehicle id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		s.renderError(w, r, http.StatusNotFound, "vehicle not found")
		return
	}
	ri := parseReferenceForm(r)
	ri.VehicleID = id
	if ri.Name == "" {
		s.setFlash(w, "Item name is required.")
		redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
		return
	}
	if _, err := s.insertReference(ri); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save item")
		return
	}
	s.setFlash(w, "Reference item added.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d#reference", id))
}

func (s *Server) editReference(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	ri, err := s.getReference(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "item not found")
		return
	}
	v, err := s.getVehicle(ri.VehicleID)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "vehicle not found")
		return
	}
	s.render(w, r, "reference_form", View{
		Title:  "Edit reference item",
		Active: "vehicles",
		Data: map[string]any{
			"Item":    ri,
			"Vehicle": v,
			"Action":  fmt.Sprintf("/reference/%d", id),
			"Kinds":   referenceKinds,
		},
	})
}

func (s *Server) updateReference(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	existing, err := s.getReference(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "item not found")
		return
	}
	ri := parseReferenceForm(r)
	ri.ID = id
	ri.VehicleID = existing.VehicleID
	if ri.Name == "" {
		s.setFlash(w, "Item name is required.")
		redirect(w, r, fmt.Sprintf("/reference/%d/edit", id))
		return
	}
	if err := s.updateReferenceRow(ri); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save item")
		return
	}
	s.setFlash(w, "Reference item updated.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d#reference", existing.VehicleID))
}

func (s *Server) deleteReference(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	ri, err := s.getReference(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "item not found")
		return
	}
	_ = s.deleteReferenceRow(id)
	s.setFlash(w, "Reference item deleted.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d#reference", ri.VehicleID))
}

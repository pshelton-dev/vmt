package handlers

import (
	"net/http"

	"vmt/internal/models"
)

// referenceInput is the JSON create/update body for a reference item.
type referenceInput struct {
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	PartNumber   string `json:"part_number"`
	Manufacturer string `json:"manufacturer"`
	Capacity     string `json:"capacity"`
	Spec         string `json:"spec"`
	Notes        string `json:"notes"`
}

func (in referenceInput) toModel() models.ReferenceItem {
	kind := strip(in.Kind)
	if kind != "fluid" {
		kind = "part"
	}
	return models.ReferenceItem{
		Kind:         kind,
		Name:         strip(in.Name),
		PartNumber:   strip(in.PartNumber),
		Manufacturer: strip(in.Manufacturer),
		Capacity:     strip(in.Capacity),
		Spec:         strip(in.Spec),
		Notes:        strip(in.Notes),
	}
}

func (s *Server) apiCreateReference(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	var in referenceInput
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	ri := in.toModel()
	ri.VehicleID = id
	if ri.Name == "" {
		apiError(w, http.StatusBadRequest, "name is required")
		return
	}
	rid, err := s.insertReference(ri)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not save item")
		return
	}
	created, err := s.getReference(rid)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load item")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) apiUpdateReference(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	existing, err := s.getReference(id)
	if err != nil {
		apiError(w, http.StatusNotFound, "item not found")
		return
	}
	var in referenceInput
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	ri := in.toModel()
	ri.ID = id
	ri.VehicleID = existing.VehicleID
	if ri.Name == "" {
		apiError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := s.updateReferenceRow(ri); err != nil {
		apiError(w, http.StatusInternalServerError, "could not save item")
		return
	}
	updated, err := s.getReference(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load item")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) apiDeleteReference(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getReference(id); err != nil {
		apiError(w, http.StatusNotFound, "item not found")
		return
	}
	if err := s.deleteReferenceRow(id); err != nil {
		apiError(w, http.StatusInternalServerError, "could not delete item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

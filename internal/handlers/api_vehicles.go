package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"vmt/internal/models"
)

// vehicleInput is the JSON create/update body for a vehicle.
type vehicleInput struct {
	Name         string  `json:"name"`
	Make         string  `json:"make"`
	Model        string  `json:"model"`
	Year         *int    `json:"year"`
	VIN          string  `json:"vin"`
	LicensePlate string  `json:"license_plate"`
	Color        string  `json:"color"`
	Odometer     int64   `json:"odometer"`
	PurchaseDate *string `json:"purchase_date"`
	Notes        string  `json:"notes"`
}

func (in vehicleInput) toModel() models.Vehicle {
	v := models.Vehicle{
		Name:         strip(in.Name),
		Make:         strip(in.Make),
		Model:        strip(in.Model),
		Year:         in.Year,
		VIN:          strip(in.VIN),
		LicensePlate: strip(in.LicensePlate),
		Color:        strip(in.Color),
		Odometer:     in.Odometer,
		Notes:        strip(in.Notes),
	}
	if in.PurchaseDate != nil {
		v.PurchaseDate = optDate(*in.PurchaseDate)
	}
	return v
}

func (s *Server) apiListVehicles(w http.ResponseWriter, r *http.Request) {
	vehicles, err := s.listVehicles()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load vehicles")
		return
	}
	reminders, _ := s.listAllReminders()
	dueByVehicle := map[int64]int{}
	for _, rem := range dueReminders(reminders) {
		dueByVehicle[rem.VehicleID]++
	}
	for i := range vehicles {
		vehicles[i].DueReminders = dueByVehicle[vehicles[i].ID]
	}
	writeJSON(w, http.StatusOK, emptyIfNil(vehicles))
}

// apiListArchivedVehicles returns the archived (no-longer-owned) vehicles.
func (s *Server) apiListArchivedVehicles(w http.ResponseWriter, r *http.Request) {
	vehicles, err := s.listArchivedVehicles()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load vehicles")
		return
	}
	writeJSON(w, http.StatusOK, emptyIfNil(vehicles))
}

// apiArchiveVehicle marks a vehicle as archived; apiUnarchiveVehicle restores
// it. Records are preserved either way.
func (s *Server) apiArchiveVehicle(w http.ResponseWriter, r *http.Request) {
	s.setArchived(w, r, true)
}

func (s *Server) apiUnarchiveVehicle(w http.ResponseWriter, r *http.Request) {
	s.setArchived(w, r, false)
}

func (s *Server) setArchived(w http.ResponseWriter, r *http.Request, archived bool) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	if err := s.setVehicleArchived(id, archived); err != nil {
		apiError(w, http.StatusInternalServerError, "could not update vehicle")
		return
	}
	updated, err := s.getVehicle(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load vehicle")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) apiCreateVehicle(w http.ResponseWriter, r *http.Request) {
	var in vehicleInput
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	v := in.toModel()
	if v.Name == "" {
		apiError(w, http.StatusBadRequest, "name is required")
		return
	}
	id, err := s.insertVehicle(v)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not save vehicle")
		return
	}
	created, err := s.getVehicle(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load vehicle")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// apiGetVehicle returns the full detail bundle the vehicle page needs in one
// request: the vehicle plus services, reminders, photos, documents and
// reference items.
func (s *Server) apiGetVehicle(w http.ResponseWriter, r *http.Request) {
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
	services, _ := s.listServices(id)
	reminders, _ := s.listRemindersByVehicle(id)
	for i := range reminders {
		annotateReminder(&reminders[i], v.Odometer)
	}
	photos, _ := s.listAttachmentsByVehicle(id, "photo")
	documents, _ := s.listAttachmentsByVehicle(id, "document")
	reference, _ := s.listReference(id)
	writeJSON(w, http.StatusOK, map[string]any{
		"vehicle":   v,
		"services":  emptyIfNil(services),
		"reminders": emptyIfNil(reminders),
		"photos":    emptyIfNil(photos),
		"documents": emptyIfNil(documents),
		"reference": emptyIfNil(reference),
	})
}

func (s *Server) apiUpdateVehicle(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	var in vehicleInput
	if err := decodeJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	v := in.toModel()
	v.ID = id
	if v.Name == "" {
		apiError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := s.updateVehicleRow(v); err != nil {
		apiError(w, http.StatusInternalServerError, "could not save vehicle")
		return
	}
	updated, err := s.getVehicle(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load vehicle")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) apiDeleteVehicle(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	names := s.storedNamesForVehicle(id)
	if err := s.deleteVehicleRow(id); err != nil {
		apiError(w, http.StatusInternalServerError, "could not delete vehicle")
		return
	}
	for _, n := range names {
		_ = os.Remove(filepath.Join(s.cfg.UploadDir, n))
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- uploads & attachments ----

// apiUploadPhoto accepts a multipart form with a "photo" file field.
func (s *Server) apiUploadPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		apiError(w, http.StatusBadRequest, "upload too large or invalid")
		return
	}
	aid, err := s.saveUpload(r, "photo", "photo", &id, nil)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not save photo")
		return
	}
	if aid == 0 {
		apiError(w, http.StatusBadRequest, "no file in \"photo\" field")
		return
	}
	// First photo becomes the primary automatically.
	if v, err := s.getVehicle(id); err == nil && v.PhotoID == nil {
		_ = s.setVehiclePhoto(id, aid)
	}
	a, err := s.getAttachment(aid)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load attachment")
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// apiUploadDocument accepts a multipart form with a "document" file field.
func (s *Server) apiUploadDocument(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	if _, err := s.getVehicle(id); err != nil {
		apiError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		apiError(w, http.StatusBadRequest, "upload too large or invalid")
		return
	}
	aid, err := s.saveUpload(r, "document", "document", &id, nil)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not save document")
		return
	}
	if aid == 0 {
		apiError(w, http.StatusBadRequest, "no file in \"document\" field")
		return
	}
	a, err := s.getAttachment(aid)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load attachment")
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// apiUploadServiceAttachment attaches a receipt/document to a service record
// (multipart "document" field).
func (s *Server) apiUploadServiceAttachment(w http.ResponseWriter, r *http.Request) {
	sid, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	sr, err := s.getService(sid)
	if err != nil {
		apiError(w, http.StatusNotFound, "service not found")
		return
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		apiError(w, http.StatusBadRequest, "upload too large or invalid")
		return
	}
	vid := sr.VehicleID
	aid, err := s.saveUpload(r, "document", "document", &vid, &sid)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not save attachment")
		return
	}
	if aid == 0 {
		apiError(w, http.StatusBadRequest, "no file in \"document\" field")
		return
	}
	a, err := s.getAttachment(aid)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load attachment")
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) apiSetPrimaryPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad vehicle id")
		return
	}
	aid, err := pathID(r, "aid")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad photo id")
		return
	}
	a, err := s.getAttachment(aid)
	if err != nil || a.VehicleID == nil || *a.VehicleID != id || a.Kind != "photo" {
		apiError(w, http.StatusNotFound, "photo not found on this vehicle")
		return
	}
	if err := s.setVehiclePhoto(id, aid); err != nil {
		apiError(w, http.StatusInternalServerError, "could not set primary photo")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiDeleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		apiError(w, http.StatusBadRequest, "bad id")
		return
	}
	a, err := s.getAttachment(id)
	if err != nil {
		apiError(w, http.StatusNotFound, "attachment not found")
		return
	}
	if err := s.deleteAttachmentRow(id); err != nil {
		apiError(w, http.StatusInternalServerError, "could not delete attachment")
		return
	}
	_ = os.Remove(filepath.Join(s.cfg.UploadDir, a.StoredName))
	w.WriteHeader(http.StatusNoContent)
}

package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"vmt/internal/models"
)

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	vehicles, err := s.listVehicles()
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not load vehicles")
		return
	}
	reminders, err := s.listAllReminders()
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not load reminders")
		return
	}
	due := dueReminders(reminders)

	// Per-vehicle due counts and grand totals.
	dueByVehicle := map[int64]int{}
	for _, rem := range due {
		dueByVehicle[rem.VehicleID]++
	}
	var totalCost float64
	var serviceCount int
	for i := range vehicles {
		vehicles[i].DueReminders = dueByVehicle[vehicles[i].ID]
		totalCost += vehicles[i].TotalCost
		serviceCount += vehicles[i].ServiceCount
	}

	recent, _ := s.recentServices(8)

	s.render(w, r, "dashboard", View{
		Title:  "Dashboard",
		Active: "dashboard",
		Data: map[string]any{
			"Vehicles":       vehicles,
			"DueReminders":   due,
			"RecentServices": recent,
			"TotalCost":      totalCost,
			"ServiceCount":   serviceCount,
		},
	})
}

func (s *Server) listVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	vehicles, err := s.listVehicles()
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not load vehicles")
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
	s.render(w, r, "vehicles", View{Title: "Vehicles", Active: "vehicles", Data: vehicles})
}

func (s *Server) newVehicle(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "vehicle_form", View{
		Title:  "Add vehicle",
		Active: "vehicles",
		Data: map[string]any{
			"Vehicle": models.Vehicle{},
			"IsNew":   true,
			"Action":  "/vehicles",
		},
	})
}

func (s *Server) createVehicle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad form")
		return
	}
	v := parseVehicleForm(r)
	if v.Name == "" {
		s.setFlash(w, "Name is required.")
		redirect(w, r, "/vehicles/new")
		return
	}
	id, err := s.insertVehicle(v)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save vehicle")
		return
	}
	if aid, _ := s.saveUpload(r, "photo", "photo", &id, nil); aid > 0 {
		_ = s.setVehiclePhoto(id, aid)
	}
	s.setFlash(w, "Vehicle added.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
}

func (s *Server) showVehicle(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	v, err := s.getVehicle(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "vehicle not found")
		return
	}
	services, _ := s.listServices(id)
	reminders, _ := s.listRemindersByVehicle(id)
	for i := range reminders {
		annotateReminder(&reminders[i], v.Odometer)
	}
	photos, _ := s.listAttachmentsByVehicle(id, "photo")
	documents, _ := s.listAttachmentsByVehicle(id, "document")

	refItems, _ := s.listReference(id)
	var parts, fluids []models.ReferenceItem
	for _, ri := range refItems {
		if ri.Kind == "fluid" {
			fluids = append(fluids, ri)
		} else {
			parts = append(parts, ri)
		}
	}

	chart := s.categoryChart(services)

	s.render(w, r, "vehicle_detail", View{
		Title:  v.Name,
		Active: "vehicles",
		Data: map[string]any{
			"Vehicle":   v,
			"Services":  services,
			"Reminders": reminders,
			"Photos":    photos,
			"Documents": documents,
			"Parts":     parts,
			"Fluids":    fluids,
			"RefKinds":  referenceKinds,
			"Chart":     chart,
		},
	})
}

func (s *Server) editVehicle(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	v, err := s.getVehicle(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "vehicle not found")
		return
	}
	s.render(w, r, "vehicle_form", View{
		Title:  "Edit " + v.Name,
		Active: "vehicles",
		Data: map[string]any{
			"Vehicle": v,
			"IsNew":   false,
			"Action":  fmt.Sprintf("/vehicles/%d", id),
		},
	})
}

func (s *Server) updateVehicle(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
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
	v := parseVehicleForm(r)
	v.ID = id
	if v.Name == "" {
		s.setFlash(w, "Name is required.")
		redirect(w, r, fmt.Sprintf("/vehicles/%d/edit", id))
		return
	}
	if err := s.updateVehicleRow(v); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save vehicle")
		return
	}
	if aid, _ := s.saveUpload(r, "photo", "photo", &id, nil); aid > 0 {
		_ = s.setVehiclePhoto(id, aid)
	}
	s.setFlash(w, "Vehicle updated.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
}

func (s *Server) deleteVehicle(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	// Remove attachment files belonging to this vehicle (directly or via services).
	names := s.storedNamesForVehicle(id)
	if err := s.deleteVehicleRow(id); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not delete vehicle")
		return
	}
	for _, n := range names {
		_ = os.Remove(filepath.Join(s.cfg.UploadDir, n))
	}
	s.setFlash(w, "Vehicle deleted.")
	redirect(w, r, "/vehicles")
}

func (s *Server) storedNamesForVehicle(id int64) []string {
	rows, err := s.db.Query(`
		SELECT stored_name FROM attachments
		WHERE vehicle_id=? OR service_id IN (SELECT id FROM service_records WHERE vehicle_id=?)`,
		id, id)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			out = append(out, n)
		}
	}
	return out
}

func parseVehicleForm(r *http.Request) models.Vehicle {
	return models.Vehicle{
		Name:         strip(r.FormValue("name")),
		Make:         strip(r.FormValue("make")),
		Model:        strip(r.FormValue("model")),
		Year:         optInt(r.FormValue("year")),
		VIN:          strip(r.FormValue("vin")),
		LicensePlate: strip(r.FormValue("license_plate")),
		Color:        strip(r.FormValue("color")),
		Odometer:     parseInt64(r.FormValue("odometer")),
		PurchaseDate: optDate(r.FormValue("purchase_date")),
		Notes:        strip(r.FormValue("notes")),
	}
}

// categoryChart builds a spending-by-category bar chart for one vehicle.
// It returns an empty value when there is nothing to show.
func (s *Server) categoryChart(services []models.ServiceRecord) template.HTML {
	totals := map[string]float64{}
	var order []string
	for _, sr := range services {
		if sr.Cost <= 0 {
			continue
		}
		if _, ok := totals[sr.Category]; !ok {
			order = append(order, sr.Category)
		}
		totals[sr.Category] += sr.Cost
	}
	if len(order) == 0 {
		return ""
	}
	sort.SliceStable(order, func(i, j int) bool { return totals[order[i]] > totals[order[j]] })
	labels := make([]string, len(order))
	values := make([]float64, len(order))
	for i, c := range order {
		labels[i] = c
		values[i] = totals[c]
	}
	cfg := s.cfg
	return barChart(labels, values, func(v float64) string { return money(cfg, v) })
}

package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"vmt/internal/models"
)

var serviceCSVHeader = []string{
	"Vehicle", "Date", "Odometer", "Category", "Description", "Vendor", "Cost", "Notes",
}

// exportAllServices streams every service record as CSV.
func (s *Server) exportAllServices(w http.ResponseWriter, r *http.Request) {
	records, err := s.allServices()
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not load records")
		return
	}
	s.writeServiceCSV(w, "vmt-services-"+time.Now().Format("20060102")+".csv", records)
}

// exportVehicleServices streams one vehicle's service records as CSV.
func (s *Server) exportVehicleServices(w http.ResponseWriter, r *http.Request) {
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
	records, err := s.listServices(id)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not load records")
		return
	}
	for i := range records {
		records[i].VehicleName = v.Name
	}
	filename := "vmt-" + slugify(v.Name) + "-" + time.Now().Format("20060102") + ".csv"
	s.writeServiceCSV(w, filename, records)
}

func (s *Server) writeServiceCSV(w http.ResponseWriter, filename string, records []models.ServiceRecord) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write(serviceCSVHeader)
	for _, sr := range records {
		odo := ""
		if sr.Odometer != nil {
			odo = strconv.FormatInt(*sr.Odometer, 10)
		}
		_ = cw.Write([]string{
			sr.VehicleName,
			sr.Date,
			odo,
			sr.Category,
			sr.Description,
			sr.Vendor,
			strconv.FormatFloat(sr.Cost, 'f', 2, 64),
			sr.Notes,
		})
	}
}

// slugify makes a filesystem/URL-friendly token from a vehicle name.
func slugify(s string) string {
	out := make([]rune, 0, len(s))
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			prevDash = false
		case r >= 'A' && r <= 'Z':
			out = append(out, r+('a'-'A'))
			prevDash = false
		default:
			if !prevDash {
				out = append(out, '-')
				prevDash = true
			}
		}
	}
	res := string(out)
	for len(res) > 0 && res[len(res)-1] == '-' {
		res = res[:len(res)-1]
	}
	for len(res) > 0 && res[0] == '-' {
		res = res[1:]
	}
	if res == "" {
		return "vehicle"
	}
	return res
}

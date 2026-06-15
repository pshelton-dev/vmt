package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"vmt/internal/models"
)

// parsedRow is one CSV data row after parsing/validation, with no DB access.
type parsedRow struct {
	Line   int
	OK     bool
	Reason string // when !OK

	// Populated when OK.
	Vehicle     string
	Date        string
	Odometer    *int64
	Category    string
	Description string
	Vendor      string
	Cost        float64
	Notes       string

	// Filled in during planning (preview): whether this row introduces a new
	// vehicle. Not used by the committer (which resolves/creates as it goes).
	NewVehicle bool
}

// importResult summarises a committed import.
type importResult struct {
	Imported        int
	VehiclesCreated int
	Skipped         int
	Issues          []string
}

// dateImportLayouts are accepted date formats, tried in order (ISO first).
var dateImportLayouts = []string{
	"2006-01-02", "2006/01/02", "01/02/2006", "1/2/2006",
	"Jan 2, 2006", "2 Jan 2006", "02 Jan 2006",
}

func parseImportDate(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	for _, l := range dateImportLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.Format("2006-01-02"), true
		}
	}
	return "", false
}

// parseImportCost strips currency symbols, thousands separators and spaces.
func parseImportCost(s string) float64 {
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' {
			b.WriteRune(r)
		}
	}
	v, _ := strconv.ParseFloat(b.String(), 64)
	return v
}

// parseServiceRows parses a service-record CSV into rows without touching the
// database. A non-nil error means the file/header is unusable (nothing to do).
func parseServiceRows(r io.Reader) ([]parsedRow, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("empty or unreadable file")
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, required := range []string{"vehicle", "date", "description"} {
		if _, ok := idx[required]; !ok {
			return nil, fmt.Errorf("missing required column %q (expected header: Vehicle, Date, Odometer, Category, Description, Vendor, Cost, Notes)", required)
		}
	}
	get := func(rec []string, name string) string {
		i, ok := idx[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	var rows []parsedRow
	line := 1
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			rows = append(rows, parsedRow{Line: line, Reason: "malformed row"})
			continue
		}
		if isBlankRecord(rec) {
			continue
		}

		name := get(rec, "vehicle")
		desc := get(rec, "description")
		date, dateOK := parseImportDate(get(rec, "date"))
		switch {
		case name == "":
			rows = append(rows, parsedRow{Line: line, Reason: "missing vehicle"})
			continue
		case desc == "":
			rows = append(rows, parsedRow{Line: line, Reason: "missing description"})
			continue
		case !dateOK:
			rows = append(rows, parsedRow{Line: line, Reason: fmt.Sprintf("unparseable date %q", get(rec, "date"))})
			continue
		}

		category := get(rec, "category")
		if category == "" {
			category = "Other"
		}
		row := parsedRow{
			Line: line, OK: true,
			Vehicle: name, Date: date, Category: category,
			Description: desc, Vendor: get(rec, "vendor"),
			Cost: parseImportCost(get(rec, "cost")), Notes: get(rec, "notes"),
		}
		row.Odometer = optInt64(strings.ReplaceAll(get(rec, "odometer"), ",", ""))
		rows = append(rows, row)
	}
	return rows, nil
}

// planImport annotates parsed rows with new-vehicle flags (read-only) and counts.
func (s *Server) planImport(rows []parsedRow) (annotated []parsedRow, newVehicles []string, wouldImport, skipped int) {
	seen := map[string]bool{}
	for _, row := range rows {
		if !row.OK {
			skipped++
			annotated = append(annotated, row)
			continue
		}
		key := strings.ToLower(row.Vehicle)
		if !seen[key] {
			if _, exists := s.vehicleIDByName(row.Vehicle); !exists {
				row.NewVehicle = true
				newVehicles = append(newVehicles, row.Vehicle)
			}
			seen[key] = true
		}
		wouldImport++
		annotated = append(annotated, row)
	}
	return
}

func (res importResult) summary() string {
	msg := fmt.Sprintf("Imported %d record(s)", res.Imported)
	if res.VehiclesCreated > 0 {
		msg += fmt.Sprintf(", created %d vehicle(s)", res.VehiclesCreated)
	}
	if res.Skipped > 0 {
		msg += fmt.Sprintf(", skipped %d row(s)", res.Skipped)
		if len(res.Issues) > 0 {
			shown := res.Issues
			if len(shown) > 3 {
				shown = shown[:3]
			}
			msg += " — " + strings.Join(shown, "; ")
			if len(res.Issues) > 3 {
				msg += fmt.Sprintf("; +%d more", len(res.Issues)-3)
			}
		}
	}
	return msg + "."
}

// previewImport parses the uploaded CSV and shows what would happen, without
// writing anything. The raw CSV is carried in the confirm form.
func (s *Server) previewImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		s.setFlash(w, "Upload too large or invalid.")
		redirect(w, r, "/settings")
		return
	}
	file, _, err := r.FormFile("csv")
	if err != nil {
		s.setFlash(w, "Choose a CSV file to import.")
		redirect(w, r, "/settings")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxUpload))
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not read file")
		return
	}

	rows, err := parseServiceRows(strings.NewReader(string(raw)))
	if err != nil {
		s.setFlash(w, "Import failed: "+err.Error())
		redirect(w, r, "/settings")
		return
	}
	annotated, newVehicles, wouldImport, skipped := s.planImport(rows)

	s.render(w, r, "import_preview", View{
		Title:  "Preview import",
		Active: "settings",
		Data: map[string]any{
			"Rows":        annotated,
			"NewVehicles": newVehicles,
			"WouldImport": wouldImport,
			"Skipped":     skipped,
			"RawCSV":      string(raw),
		},
	})
}

// importData commits an import. It accepts the raw CSV from the preview confirm
// form (csv_data) or, as a fallback, a freshly uploaded file (csv).
func (s *Server) importData(w http.ResponseWriter, r *http.Request) {
	// r.FormValue parses either a urlencoded body (the preview "Confirm" form,
	// carrying csv_data) or a multipart body (a direct file upload).
	var reader io.Reader
	if data := r.FormValue("csv_data"); data != "" {
		reader = strings.NewReader(data)
	} else if file, _, err := r.FormFile("csv"); err == nil {
		defer file.Close()
		reader = io.LimitReader(file, maxUpload)
	} else {
		s.setFlash(w, "Choose a CSV file to import.")
		redirect(w, r, "/settings")
		return
	}

	rows, err := parseServiceRows(reader)
	if err != nil {
		s.setFlash(w, "Import failed: "+err.Error())
		redirect(w, r, "/settings")
		return
	}

	var res importResult
	cache := map[string]int64{}
	for _, row := range rows {
		if !row.OK {
			res.Skipped++
			res.Issues = append(res.Issues, fmt.Sprintf("line %d: %s", row.Line, row.Reason))
			continue
		}
		vehID, err := s.resolveVehicle(row.Vehicle, cache, &res)
		if err != nil {
			res.Skipped++
			res.Issues = append(res.Issues, fmt.Sprintf("line %d: could not create vehicle", row.Line))
			continue
		}
		sr := models.ServiceRecord{
			VehicleID: vehID, Date: row.Date, Odometer: row.Odometer,
			Category: row.Category, Description: row.Description,
			Vendor: row.Vendor, Cost: row.Cost, Notes: row.Notes,
		}
		if _, err := s.insertService(sr); err != nil {
			res.Skipped++
			res.Issues = append(res.Issues, fmt.Sprintf("line %d: database error", row.Line))
			continue
		}
		if sr.Odometer != nil {
			s.bumpOdometer(vehID, *sr.Odometer)
		}
		res.Imported++
	}

	s.setFlash(w, res.summary())
	redirect(w, r, "/settings")
}

// resolveVehicle returns the id for a vehicle name, creating it if necessary.
func (s *Server) resolveVehicle(name string, cache map[string]int64, res *importResult) (int64, error) {
	key := strings.ToLower(name)
	if id, ok := cache[key]; ok {
		return id, nil
	}
	if id, ok := s.vehicleIDByName(name); ok {
		cache[key] = id
		return id, nil
	}
	id, err := s.insertVehicle(models.Vehicle{Name: name})
	if err != nil {
		return 0, err
	}
	cache[key] = id
	res.VehiclesCreated++
	return id, nil
}

func isBlankRecord(rec []string) bool {
	for _, f := range rec {
		if strings.TrimSpace(f) != "" {
			return false
		}
	}
	return true
}

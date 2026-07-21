package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vmt/internal/auth"
	"vmt/internal/config"
	"vmt/internal/db"
)

const testPassword = "test-password"

// newTestAPI builds a Server over a temp database, sets the admin password,
// and returns the routed handler plus a valid session cookie.
func newTestAPI(t *testing.T) (http.Handler, *http.Cookie) {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(filepath.Join(dir, "vmt.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	am := auth.New(d, time.Hour)
	if err := am.SetPassword(testPassword); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		DataDir:      dir,
		DBPath:       filepath.Join(dir, "vmt.db"),
		UploadDir:    filepath.Join(dir, "uploads"),
		Currency:     "$",
		DistanceUnit: "mi",
		DateFormat:   "Jan 2, 2006",
	}
	s, err := New(d, am, cfg)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := am.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	return s.Routes(), &http.Cookie{Name: auth.CookieName(), Value: token}
}

// call sends a request (JSON body if body != nil) and returns the recorder.
func call(t *testing.T, h http.Handler, cookie *http.Cookie, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// decode unmarshals a recorder's JSON body into v.
func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("bad JSON response (%d): %v\n%s", w.Code, err, w.Body.String())
	}
}

func want(t *testing.T, w *httptest.ResponseRecorder, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("want HTTP %d, got %d: %s", code, w.Code, w.Body.String())
	}
}

// makeVehicle creates a vehicle and returns its id.
func makeVehicle(t *testing.T, h http.Handler, c *http.Cookie, name string) int64 {
	t.Helper()
	w := call(t, h, c, "POST", "/api/v1/vehicles", map[string]any{"name": name, "odometer": 50000})
	want(t, w, http.StatusCreated)
	var v struct {
		ID int64 `json:"id"`
	}
	decode(t, w, &v)
	return v.ID
}

// ---- auth ----

func TestAPIAuth(t *testing.T) {
	h, cookie := newTestAPI(t)

	// Unauthenticated protected call -> 401 JSON.
	w := call(t, h, nil, "GET", "/api/v1/vehicles", nil)
	want(t, w, http.StatusUnauthorized)
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("401 should be JSON, got %q", ct)
	}

	// Session status.
	w = call(t, h, nil, "GET", "/api/v1/session", nil)
	want(t, w, http.StatusOK)
	var sess map[string]bool
	decode(t, w, &sess)
	if !sess["configured"] || sess["authed"] {
		t.Fatalf("expected configured && !authed, got %v", sess)
	}

	// Wrong password.
	w = call(t, h, nil, "POST", "/api/v1/login", map[string]string{"password": "nope"})
	want(t, w, http.StatusUnauthorized)

	// Right password sets a session cookie that works.
	w = call(t, h, nil, "POST", "/api/v1/login", map[string]string{"password": testPassword})
	want(t, w, http.StatusNoContent)
	res := w.Result()
	if len(res.Cookies()) == 0 {
		t.Fatal("login did not set a session cookie")
	}
	w = call(t, h, res.Cookies()[0], "GET", "/api/v1/vehicles", nil)
	want(t, w, http.StatusOK)

	// Logout invalidates it.
	w = call(t, h, res.Cookies()[0], "POST", "/api/v1/logout", nil)
	want(t, w, http.StatusNoContent)
	w = call(t, h, res.Cookies()[0], "GET", "/api/v1/vehicles", nil)
	want(t, w, http.StatusUnauthorized)

	_ = cookie
}

// ---- vehicles ----

func TestAPIVehiclesCRUD(t *testing.T) {
	h, c := newTestAPI(t)

	// Missing name rejected.
	w := call(t, h, c, "POST", "/api/v1/vehicles", map[string]any{"name": "  "})
	want(t, w, http.StatusBadRequest)

	id := makeVehicle(t, h, c, "Truck")

	// List has one, JSON array (not null).
	w = call(t, h, c, "GET", "/api/v1/vehicles", nil)
	want(t, w, http.StatusOK)
	var list []map[string]any
	decode(t, w, &list)
	if len(list) != 1 || list[0]["name"] != "Truck" {
		t.Fatalf("unexpected list: %v", list)
	}

	// Detail bundle shape.
	w = call(t, h, c, "GET", fmt.Sprintf("/api/v1/vehicles/%d", id), nil)
	want(t, w, http.StatusOK)
	var detail map[string]json.RawMessage
	decode(t, w, &detail)
	for _, k := range []string{"vehicle", "services", "reminders", "photos", "documents", "reference"} {
		if _, ok := detail[k]; !ok {
			t.Fatalf("detail bundle missing %q", k)
		}
	}

	// Update.
	w = call(t, h, c, "PUT", fmt.Sprintf("/api/v1/vehicles/%d", id),
		map[string]any{"name": "Truck", "color": "Red", "year": 2020, "odometer": 51000})
	want(t, w, http.StatusOK)
	var upd map[string]any
	decode(t, w, &upd)
	if upd["color"] != "Red" || upd["year"].(float64) != 2020 {
		t.Fatalf("update not reflected: %v", upd)
	}

	// Delete, then 404.
	w = call(t, h, c, "DELETE", fmt.Sprintf("/api/v1/vehicles/%d", id), nil)
	want(t, w, http.StatusNoContent)
	w = call(t, h, c, "GET", fmt.Sprintf("/api/v1/vehicles/%d", id), nil)
	want(t, w, http.StatusNotFound)
}

// TestAPIVehicleArchive covers archiving a vehicle no longer owned: it leaves
// the active list, dashboard totals and reminders, but keeps its records and
// still counts in the (historical) reports.
func TestAPIVehicleArchive(t *testing.T) {
	h, c := newTestAPI(t)
	kept := makeVehicle(t, h, c, "Keeper")
	sold := makeVehicle(t, h, c, "Sold")

	// $100 of work on each, and a due reminder on the one we'll archive.
	for _, id := range []int64{kept, sold} {
		w := call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/services", id), map[string]any{
			"date": time.Now().Format("2006-01-02"), "description": "work",
			"category": "Repair", "cost": 100,
		})
		want(t, w, http.StatusCreated)
	}
	w := call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/reminders", sold), map[string]any{
		"title": "Overdue thing", "due_date": time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
	})
	want(t, w, http.StatusCreated)

	// Archive it.
	w = call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/archive", sold), nil)
	want(t, w, http.StatusOK)
	var arch map[string]any
	decode(t, w, &arch)
	if arch["archived"] != true {
		t.Fatalf("expected archived=true, got %v", arch["archived"])
	}

	// Gone from the active list, present in the archived list.
	w = call(t, h, c, "GET", "/api/v1/vehicles", nil)
	want(t, w, http.StatusOK)
	var active []map[string]any
	decode(t, w, &active)
	if len(active) != 1 || active[0]["name"] != "Keeper" {
		t.Fatalf("active list should hold only Keeper, got %v", active)
	}
	w = call(t, h, c, "GET", "/api/v1/vehicles/archived", nil)
	want(t, w, http.StatusOK)
	var archived []map[string]any
	decode(t, w, &archived)
	if len(archived) != 1 || archived[0]["name"] != "Sold" {
		t.Fatalf("archived list should hold only Sold, got %v", archived)
	}

	// Its records survive and are still reachable.
	w = call(t, h, c, "GET", fmt.Sprintf("/api/v1/vehicles/%d", sold), nil)
	want(t, w, http.StatusOK)
	var detail struct {
		Services []map[string]any `json:"services"`
	}
	decode(t, w, &detail)
	if len(detail.Services) != 1 {
		t.Fatalf("archived vehicle should keep its records, got %d", len(detail.Services))
	}

	// Dashboard drops it from the running totals and the due reminders.
	w = call(t, h, c, "GET", "/api/v1/dashboard", nil)
	want(t, w, http.StatusOK)
	var dash struct {
		TotalCost      float64          `json:"total_cost"`
		ServiceCount   int              `json:"service_count"`
		DueReminders   []map[string]any `json:"due_reminders"`
		Vehicles       []map[string]any `json:"vehicles"`
		RecentServices []map[string]any `json:"recent_services"`
	}
	decode(t, w, &dash)
	if dash.TotalCost != 100 || dash.ServiceCount != 1 {
		t.Fatalf("dashboard should exclude archived: %+v", dash)
	}
	if len(dash.Vehicles) != 1 || len(dash.DueReminders) != 0 {
		t.Fatalf("dashboard should drop archived vehicle and its reminders: %+v", dash)
	}
	// The recent feed must agree with the counts beside it.
	if len(dash.RecentServices) != 1 || dash.RecentServices[0]["vehicle_name"] != "Keeper" {
		t.Fatalf("recent services should exclude archived vehicles: %v", dash.RecentServices)
	}

	// Reports stay historical — both vehicles' spending still counts.
	w = call(t, h, c, "GET", "/api/v1/reports", nil)
	want(t, w, http.StatusOK)
	var rep struct {
		TotalCost float64          `json:"total_cost"`
		ByVehicle []map[string]any `json:"by_vehicle"`
	}
	decode(t, w, &rep)
	if rep.TotalCost != 200 || len(rep.ByVehicle) != 2 {
		t.Fatalf("reports should still include archived: %+v", rep)
	}

	// Unarchiving restores it.
	w = call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/unarchive", sold), nil)
	want(t, w, http.StatusOK)
	w = call(t, h, c, "GET", "/api/v1/vehicles", nil)
	want(t, w, http.StatusOK)
	decode(t, w, &active)
	if len(active) != 2 {
		t.Fatalf("unarchive should restore the vehicle, got %v", active)
	}
}

// ---- services ----

func TestAPIServices(t *testing.T) {
	h, c := newTestAPI(t)
	id := makeVehicle(t, h, c, "Car")

	// Validation.
	w := call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/services", id),
		map[string]any{"description": "Oil change"})
	want(t, w, http.StatusBadRequest)

	// Create with a higher odometer bumps the vehicle.
	w = call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/services", id), map[string]any{
		"date": "2026-07-01", "description": "Oil change", "category": "Oil Change",
		"odometer": 55000, "cost": 89.5,
	})
	want(t, w, http.StatusCreated)
	var sr map[string]any
	decode(t, w, &sr)
	sid := int64(sr["id"].(float64))

	w = call(t, h, c, "GET", fmt.Sprintf("/api/v1/vehicles/%d", id), nil)
	var detail struct {
		Vehicle struct {
			Odometer int64 `json:"odometer"`
		} `json:"vehicle"`
		Services []map[string]any `json:"services"`
	}
	decode(t, w, &detail)
	if detail.Vehicle.Odometer != 55000 {
		t.Fatalf("odometer not bumped: %d", detail.Vehicle.Odometer)
	}
	if len(detail.Services) != 1 {
		t.Fatalf("want 1 service, got %d", len(detail.Services))
	}

	// Update + delete.
	w = call(t, h, c, "PUT", fmt.Sprintf("/api/v1/services/%d", sid), map[string]any{
		"date": "2026-07-01", "description": "Oil + filter", "cost": 99.0,
	})
	want(t, w, http.StatusOK)
	w = call(t, h, c, "DELETE", fmt.Sprintf("/api/v1/services/%d", sid), nil)
	want(t, w, http.StatusNoContent)
	w = call(t, h, c, "PUT", fmt.Sprintf("/api/v1/services/%d", sid),
		map[string]any{"date": "2026-07-01", "description": "x"})
	want(t, w, http.StatusNotFound)
}

// ---- reminders ----

func TestAPIReminders(t *testing.T) {
	h, c := newTestAPI(t)
	id := makeVehicle(t, h, c, "Bike")

	// Overdue by date.
	w := call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/reminders", id), map[string]any{
		"title": "Chain lube", "due_date": "2026-01-01",
	})
	want(t, w, http.StatusCreated)
	var rem map[string]any
	decode(t, w, &rem)
	if rem["status"] != "overdue" {
		t.Fatalf("want overdue status, got %v", rem["status"])
	}

	// Recurring completion rolls a new one.
	w = call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/reminders", id), map[string]any{
		"title": "Oil change", "due_date": "2026-01-01", "interval_months": 6,
	})
	want(t, w, http.StatusCreated)
	decode(t, w, &rem)
	rid := int64(rem["id"].(float64))
	w = call(t, h, c, "POST", fmt.Sprintf("/api/v1/reminders/%d/complete", rid), nil)
	want(t, w, http.StatusNoContent)

	w = call(t, h, c, "GET", "/api/v1/reminders", nil)
	want(t, w, http.StatusOK)
	var list []map[string]any
	decode(t, w, &list)
	// Chain lube + the rolled Oil change (the completed one is filtered out).
	if len(list) != 2 {
		t.Fatalf("want 2 active reminders, got %d: %v", len(list), list)
	}
}

// ---- reference ----

func TestAPIReference(t *testing.T) {
	h, c := newTestAPI(t)
	id := makeVehicle(t, h, c, "Tractor")

	w := call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/reference", id), map[string]any{
		"kind": "part", "name": "Oil filter", "part_number": "HH150-32094", "manufacturer": "Kubota",
	})
	want(t, w, http.StatusCreated)
	var ri map[string]any
	decode(t, w, &ri)
	rid := int64(ri["id"].(float64))

	w = call(t, h, c, "PUT", fmt.Sprintf("/api/v1/reference/%d", rid), map[string]any{
		"kind": "part", "name": "Oil filter", "part_number": "HH160-32094",
	})
	want(t, w, http.StatusOK)
	decode(t, w, &ri)
	if ri["part_number"] != "HH160-32094" {
		t.Fatalf("update not reflected: %v", ri)
	}

	w = call(t, h, c, "DELETE", fmt.Sprintf("/api/v1/reference/%d", rid), nil)
	want(t, w, http.StatusNoContent)
}

// ---- reports ----

func TestAPIReports(t *testing.T) {
	h, c := newTestAPI(t)
	id := makeVehicle(t, h, c, "Car")
	for i, cost := range []float64{100, 50} {
		w := call(t, h, c, "POST", fmt.Sprintf("/api/v1/vehicles/%d/services", id), map[string]any{
			"date":        time.Now().AddDate(0, -i, 0).Format("2006-01-02"),
			"description": "work", "category": "Repair", "cost": cost,
		})
		want(t, w, http.StatusCreated)
	}
	w := call(t, h, c, "GET", "/api/v1/reports", nil)
	want(t, w, http.StatusOK)
	var rep struct {
		TotalCost    float64          `json:"total_cost"`
		ServiceCount int              `json:"service_count"`
		ByVehicle    []map[string]any `json:"by_vehicle"`
		Monthly      []map[string]any `json:"monthly"`
	}
	decode(t, w, &rep)
	if rep.TotalCost != 150 || rep.ServiceCount != 2 {
		t.Fatalf("bad totals: %+v", rep)
	}
	if len(rep.ByVehicle) != 1 || rep.ByVehicle[0]["total"].(float64) != 150 {
		t.Fatalf("bad by_vehicle: %v", rep.ByVehicle)
	}
	if len(rep.Monthly) != 12 {
		t.Fatalf("want 12 monthly buckets, got %d", len(rep.Monthly))
	}
}

// ---- settings ----

func TestAPISettings(t *testing.T) {
	h, c := newTestAPI(t)

	w := call(t, h, c, "PUT", "/api/v1/settings", map[string]any{
		"currency": "€", "date_format": "2006-01-02", "notify_enabled": true, "notify_email": "me@example.com",
	})
	want(t, w, http.StatusOK)
	var st map[string]any
	decode(t, w, &st)
	if st["currency"] != "€" || st["notify_email"] != "me@example.com" || st["notify_enabled"] != true {
		t.Fatalf("settings not saved: %v", st)
	}

	// Invalid date format rejected.
	w = call(t, h, c, "PUT", "/api/v1/settings", map[string]any{"date_format": "bogus"})
	want(t, w, http.StatusBadRequest)

	// Password change: wrong current -> 403; right -> new password logs in.
	w = call(t, h, c, "POST", "/api/v1/settings/password", map[string]string{"current": "wrong", "new": "newpass123"})
	want(t, w, http.StatusForbidden)
	w = call(t, h, c, "POST", "/api/v1/settings/password", map[string]string{"current": testPassword, "new": "newpass123"})
	want(t, w, http.StatusNoContent)
	w = call(t, h, nil, "POST", "/api/v1/login", map[string]string{"password": "newpass123"})
	want(t, w, http.StatusNoContent)
}

// ---- import ----

const importCSV = "Vehicle,Date,Odometer,Category,Description,Vendor,Cost,Notes\n" +
	"NewCar,2026-06-01,1000,Oil Change,First oil change,Shop,50.00,\n" +
	"NewCar,bad-date,,,broken row,,,\n"

func TestAPIImport(t *testing.T) {
	h, c := newTestAPI(t)

	// Preview: reports plan, writes nothing.
	w := call(t, h, c, "POST", "/api/v1/import/preview", map[string]string{"csv_data": importCSV})
	want(t, w, http.StatusOK)
	var prev struct {
		WouldImport int      `json:"would_import"`
		Skipped     int      `json:"skipped"`
		NewVehicles []string `json:"new_vehicles"`
		CSVData     string   `json:"csv_data"`
	}
	decode(t, w, &prev)
	if prev.WouldImport != 1 || prev.Skipped != 1 || len(prev.NewVehicles) != 1 {
		t.Fatalf("bad preview: %+v", prev)
	}
	w = call(t, h, c, "GET", "/api/v1/vehicles", nil)
	var list []map[string]any
	decode(t, w, &list)
	if len(list) != 0 {
		t.Fatal("preview must not create vehicles")
	}

	// Commit: creates the vehicle and record.
	w = call(t, h, c, "POST", "/api/v1/import", map[string]string{"csv_data": prev.CSVData})
	want(t, w, http.StatusOK)
	var res struct {
		Imported        int `json:"imported"`
		VehiclesCreated int `json:"vehicles_created"`
		Skipped         int `json:"skipped"`
	}
	decode(t, w, &res)
	if res.Imported != 1 || res.VehiclesCreated != 1 || res.Skipped != 1 {
		t.Fatalf("bad import result: %+v", res)
	}
	w = call(t, h, c, "GET", "/api/v1/vehicles", nil)
	decode(t, w, &list)
	if len(list) != 1 || list[0]["service_count"].(float64) != 1 {
		t.Fatalf("import not committed: %v", list)
	}
}

// ---- uploads ----

func TestAPIPhotoUpload(t *testing.T) {
	h, c := newTestAPI(t)
	id := makeVehicle(t, h, c, "Car")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("photo", "front.jpg")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("fake-jpeg-bytes"))
	mw.Close()

	r := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/vehicles/%d/photos", id), &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.AddCookie(c)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	want(t, w, http.StatusCreated)

	var a map[string]any
	decode(t, w, &a)
	if a["original_name"] != "front.jpg" || a["kind"] != "photo" {
		t.Fatalf("bad attachment: %v", a)
	}
	// First photo becomes primary.
	w2 := call(t, h, c, "GET", fmt.Sprintf("/api/v1/vehicles/%d", id), nil)
	var detail struct {
		Vehicle struct {
			PhotoID *int64 `json:"photo_id"`
		} `json:"vehicle"`
	}
	decode(t, w2, &detail)
	if detail.Vehicle.PhotoID == nil {
		t.Fatal("first photo should be set as primary")
	}
	// The stored file is downloadable.
	aid := int64(a["id"].(float64))
	w3 := call(t, h, c, "GET", fmt.Sprintf("/api/v1/files/%d", aid), nil)
	want(t, w3, http.StatusOK)
	if w3.Body.String() != "fake-jpeg-bytes" {
		t.Fatal("served file does not match upload")
	}
}

// Package models holds the domain types shared across the app.
package models

import "time"

type Vehicle struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Make         string  `json:"make"`
	Model        string  `json:"model"`
	Year         *int    `json:"year"`
	VIN          string  `json:"vin"`
	LicensePlate string  `json:"license_plate"`
	Color        string  `json:"color"`
	Odometer     int64   `json:"odometer"`
	PurchaseDate *string `json:"purchase_date"` // ISO yyyy-mm-dd
	Notes        string  `json:"notes"`
	PhotoID      *int64  `json:"photo_id"`
	CreatedAt    time.Time `json:"-"`

	// Derived / joined fields (not stored on the vehicles row).
	TotalCost    float64 `json:"total_cost"`
	ServiceCount int     `json:"service_count"`
	DueReminders int     `json:"due_reminders"`
}

type ServiceRecord struct {
	ID          int64   `json:"id"`
	VehicleID   int64   `json:"vehicle_id"`
	Date        string  `json:"date"` // ISO yyyy-mm-dd
	Odometer    *int64  `json:"odometer"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Vendor      string  `json:"vendor"`
	Cost        float64 `json:"cost"`
	Notes       string  `json:"notes"`
	CreatedAt   time.Time `json:"-"`

	VehicleName string       `json:"vehicle_name,omitempty"` // joined, for cross-vehicle listings
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Reminder struct {
	ID             int64   `json:"id"`
	VehicleID      int64   `json:"vehicle_id"`
	Title          string  `json:"title"`
	DueDate        *string `json:"due_date"`
	DueOdometer    *int64  `json:"due_odometer"`
	IntervalMonths *int    `json:"interval_months"`
	IntervalMiles  *int    `json:"interval_miles"`
	Notes          string  `json:"notes"`
	Completed      bool    `json:"completed"`
	Notify         bool    `json:"notify"`
	LastNotified   *string `json:"last_notified"`
	CreatedAt      time.Time `json:"-"`

	// Derived status for display.
	VehicleName string `json:"vehicle_name,omitempty"`
	Status      string `json:"status,omitempty"`      // "ok", "soon", "due", "overdue"
	StatusText  string `json:"status_text,omitempty"`
}

// ReferenceItem is a per-vehicle quick-reference spec: a part (filter, plug,
// belt) with part number/manufacturer, or a fluid with capacity and grade.
type ReferenceItem struct {
	ID           int64  `json:"id"`
	VehicleID    int64  `json:"vehicle_id"`
	Kind         string `json:"kind"` // "part" or "fluid"
	Name         string `json:"name"`
	PartNumber   string `json:"part_number"`
	Manufacturer string `json:"manufacturer"`
	Capacity     string `json:"capacity"`
	Spec         string `json:"spec"`
	Notes        string `json:"notes"`
	Position     int    `json:"position"`
	CreatedAt    time.Time `json:"-"`
}

type Attachment struct {
	ID           int64  `json:"id"`
	VehicleID    *int64 `json:"vehicle_id"`
	ServiceID    *int64 `json:"service_id"`
	Kind         string `json:"kind"` // "photo" or "document"
	StoredName   string `json:"-"`
	OriginalName string `json:"original_name"`
	ContentType  string `json:"content_type"`
	Size         int64  `json:"size"`
	CreatedAt    time.Time `json:"-"`
}

// ServiceCategories are the selectable categories for a service record.
var ServiceCategories = []string{
	"Oil Change",
	"Tires",
	"Brakes",
	"Battery",
	"Engine",
	"Transmission",
	"Inspection",
	"Registration",
	"Insurance",
	"Repair",
	"Detailing",
	"Other",
}

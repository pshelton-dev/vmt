// Package models holds the domain types shared across the app.
package models

import "time"

type Vehicle struct {
	ID           int64
	Name         string
	Make         string
	Model        string
	Year         *int
	VIN          string
	LicensePlate string
	Color        string
	Odometer     int64
	PurchaseDate *string // ISO yyyy-mm-dd
	Notes        string
	PhotoID      *int64
	CreatedAt    time.Time

	// Derived / joined fields (not stored on the vehicles row).
	TotalCost    float64
	ServiceCount int
	DueReminders int
}

type ServiceRecord struct {
	ID          int64
	VehicleID   int64
	Date        string // ISO yyyy-mm-dd
	Odometer    *int64
	Category    string
	Description string
	Vendor      string
	Cost        float64
	Notes       string
	CreatedAt   time.Time

	VehicleName string // joined, for cross-vehicle listings
	Attachments []Attachment
}

type Reminder struct {
	ID             int64
	VehicleID      int64
	Title          string
	DueDate        *string
	DueOdometer    *int64
	IntervalMonths *int
	IntervalMiles  *int
	Notes          string
	Completed      bool
	Notify         bool
	LastNotified   *string
	CreatedAt      time.Time

	// Derived status for display.
	VehicleName string
	Status      string // "ok", "soon", "due", "overdue"
	StatusText  string
}

// ReferenceItem is a per-vehicle quick-reference spec: a part (filter, plug,
// belt) with part number/manufacturer, or a fluid with capacity and grade.
type ReferenceItem struct {
	ID           int64
	VehicleID    int64
	Kind         string // "part" or "fluid"
	Name         string
	PartNumber   string
	Manufacturer string
	Capacity     string
	Spec         string
	Notes        string
	Position     int
	CreatedAt    time.Time
}

type Attachment struct {
	ID           int64
	VehicleID    *int64
	ServiceID    *int64
	Kind         string // "photo" or "document"
	StoredName   string
	OriginalName string
	ContentType  string
	Size         int64
	CreatedAt    time.Time
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

package handlers

import (
	"database/sql"

	"vmt/internal/models"
)

// ---- nullable scan helpers ----

func nullInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullIntp(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
func nullStr(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func toI64(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}
func toInt(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}
func toStr(n sql.NullString) *string {
	if !n.Valid || n.String == "" {
		return nil
	}
	v := n.String
	return &v
}

// ---- vehicles ----

// listVehicles returns the actively-owned vehicles (archived ones excluded)
// with derived totals, ordered by name.
func (s *Server) listVehicles() ([]models.Vehicle, error) {
	return s.vehiclesWhere("v.archived = 0")
}

// listArchivedVehicles returns vehicles that have been archived (no longer
// owned) with derived totals, ordered by name.
func (s *Server) listArchivedVehicles() ([]models.Vehicle, error) {
	return s.vehiclesWhere("v.archived = 1")
}

// vehiclesWhere runs the vehicle-list query with the given WHERE predicate.
func (s *Server) vehiclesWhere(where string) ([]models.Vehicle, error) {
	rows, err := s.db.Query(`
		SELECT v.id, v.name, v.make, v.model, v.year, v.vin, v.license_plate,
		       v.color, v.odometer, v.purchase_date, v.notes, v.photo_id, v.archived,
		       COALESCE(sc.cnt,0), COALESCE(sc.total,0)
		FROM vehicles v
		LEFT JOIN (
			SELECT vehicle_id, COUNT(*) cnt, SUM(cost) total
			FROM service_records GROUP BY vehicle_id
		) sc ON sc.vehicle_id = v.id
		WHERE ` + where + `
		ORDER BY v.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Vehicle
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Server) getVehicle(id int64) (models.Vehicle, error) {
	row := s.db.QueryRow(`
		SELECT v.id, v.name, v.make, v.model, v.year, v.vin, v.license_plate,
		       v.color, v.odometer, v.purchase_date, v.notes, v.photo_id, v.archived,
		       COALESCE(sc.cnt,0), COALESCE(sc.total,0)
		FROM vehicles v
		LEFT JOIN (
			SELECT vehicle_id, COUNT(*) cnt, SUM(cost) total
			FROM service_records GROUP BY vehicle_id
		) sc ON sc.vehicle_id = v.id
		WHERE v.id = ?`, id)
	return scanVehicle(row)
}

type scanner interface{ Scan(...any) error }

func scanVehicle(r scanner) (models.Vehicle, error) {
	var v models.Vehicle
	var year, photoID sql.NullInt64
	var purchase sql.NullString
	var archived int
	err := r.Scan(&v.ID, &v.Name, &v.Make, &v.Model, &year, &v.VIN, &v.LicensePlate,
		&v.Color, &v.Odometer, &purchase, &v.Notes, &photoID, &archived,
		&v.ServiceCount, &v.TotalCost)
	if err != nil {
		return v, err
	}
	v.Year = toInt(year)
	v.PhotoID = toI64(photoID)
	v.PurchaseDate = toStr(purchase)
	v.Archived = archived != 0
	return v, nil
}

func (s *Server) insertVehicle(v models.Vehicle) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO vehicles(name, make, model, year, vin, license_plate, color, odometer, purchase_date, notes)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		v.Name, v.Make, v.Model, nullIntp(v.Year), v.VIN, v.LicensePlate, v.Color,
		v.Odometer, nullStr(v.PurchaseDate), v.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Server) updateVehicleRow(v models.Vehicle) error {
	_, err := s.db.Exec(`
		UPDATE vehicles SET name=?, make=?, model=?, year=?, vin=?, license_plate=?,
		       color=?, odometer=?, purchase_date=?, notes=? WHERE id=?`,
		v.Name, v.Make, v.Model, nullIntp(v.Year), v.VIN, v.LicensePlate, v.Color,
		v.Odometer, nullStr(v.PurchaseDate), v.Notes, v.ID)
	return err
}

func (s *Server) deleteVehicleRow(id int64) error {
	_, err := s.db.Exec(`DELETE FROM vehicles WHERE id=?`, id)
	return err
}

// vehicleIDByName returns the id of a vehicle whose name matches (case-
// insensitively), and whether one was found.
func (s *Server) vehicleIDByName(name string) (int64, bool) {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM vehicles WHERE name = ? COLLATE NOCASE ORDER BY id LIMIT 1`, name,
	).Scan(&id)
	if err != nil {
		return 0, false
	}
	return id, true
}

// setVehicleArchived flags a vehicle as archived (no longer owned) or restores
// it. Records are untouched; the flag only controls where it's listed and
// whether it counts toward active fleet totals and reminders.
func (s *Server) setVehicleArchived(id int64, archived bool) error {
	_, err := s.db.Exec(`UPDATE vehicles SET archived=? WHERE id=?`, boolInt(archived), id)
	return err
}

func (s *Server) setVehiclePhoto(vehicleID, photoID int64) error {
	_, err := s.db.Exec(`UPDATE vehicles SET photo_id=? WHERE id=?`, photoID, vehicleID)
	return err
}

// bumpOdometer raises a vehicle's recorded odometer if the new value is higher.
func (s *Server) bumpOdometer(vehicleID, odo int64) {
	if odo <= 0 {
		return
	}
	_, _ = s.db.Exec(`UPDATE vehicles SET odometer=? WHERE id=? AND odometer < ?`, odo, vehicleID, odo)
}

// ---- service records ----

func (s *Server) listServices(vehicleID int64) ([]models.ServiceRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, vehicle_id, date, odometer, category, description, vendor, cost, notes
		FROM service_records WHERE vehicle_id=? ORDER BY date DESC, id DESC`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ServiceRecord
	for rows.Next() {
		sr, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// attach counts of attachments per record
	for i := range out {
		atts, _ := s.listAttachmentsByService(out[i].ID)
		out[i].Attachments = atts
	}
	return out, nil
}

func (s *Server) getService(id int64) (models.ServiceRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, vehicle_id, date, odometer, category, description, vendor, cost, notes
		FROM service_records WHERE id=?`, id)
	return scanService(row)
}

func scanService(r scanner) (models.ServiceRecord, error) {
	var sr models.ServiceRecord
	var odo sql.NullInt64
	err := r.Scan(&sr.ID, &sr.VehicleID, &sr.Date, &odo, &sr.Category,
		&sr.Description, &sr.Vendor, &sr.Cost, &sr.Notes)
	if err != nil {
		return sr, err
	}
	sr.Odometer = toI64(odo)
	return sr, nil
}

func (s *Server) insertService(sr models.ServiceRecord) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO service_records(vehicle_id, date, odometer, category, description, vendor, cost, notes)
		VALUES(?,?,?,?,?,?,?,?)`,
		sr.VehicleID, sr.Date, nullInt(sr.Odometer), sr.Category, sr.Description, sr.Vendor, sr.Cost, sr.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Server) updateServiceRow(sr models.ServiceRecord) error {
	_, err := s.db.Exec(`
		UPDATE service_records SET date=?, odometer=?, category=?, description=?, vendor=?, cost=?, notes=?
		WHERE id=?`,
		sr.Date, nullInt(sr.Odometer), sr.Category, sr.Description, sr.Vendor, sr.Cost, sr.Notes, sr.ID)
	return err
}

func (s *Server) deleteServiceRow(id int64) error {
	_, err := s.db.Exec(`DELETE FROM service_records WHERE id=?`, id)
	return err
}

// allServices returns every service record across all vehicles, with the
// vehicle name joined, ordered for a stable export.
func (s *Server) allServices() ([]models.ServiceRecord, error) {
	rows, err := s.db.Query(`
		SELECT sr.id, sr.vehicle_id, sr.date, sr.odometer, sr.category, sr.description,
		       sr.vendor, sr.cost, sr.notes, v.name
		FROM service_records sr JOIN vehicles v ON v.id = sr.vehicle_id
		ORDER BY v.name COLLATE NOCASE, sr.date DESC, sr.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ServiceRecord
	for rows.Next() {
		var sr models.ServiceRecord
		var odo sql.NullInt64
		if err := rows.Scan(&sr.ID, &sr.VehicleID, &sr.Date, &odo, &sr.Category,
			&sr.Description, &sr.Vendor, &sr.Cost, &sr.Notes, &sr.VehicleName); err != nil {
			return nil, err
		}
		sr.Odometer = toI64(odo)
		out = append(out, sr)
	}
	return out, rows.Err()
}

// recentServices returns the most recent records across the actively-owned
// vehicles. Archived ones are left out so the dashboard feed matches the
// counts and totals beside it.
func (s *Server) recentServices(limit int) ([]models.ServiceRecord, error) {
	rows, err := s.db.Query(`
		SELECT sr.id, sr.vehicle_id, sr.date, sr.odometer, sr.category, sr.description,
		       sr.vendor, sr.cost, sr.notes, v.name
		FROM service_records sr JOIN vehicles v ON v.id = sr.vehicle_id
		WHERE v.archived = 0
		ORDER BY sr.date DESC, sr.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ServiceRecord
	for rows.Next() {
		var sr models.ServiceRecord
		var odo sql.NullInt64
		if err := rows.Scan(&sr.ID, &sr.VehicleID, &sr.Date, &odo, &sr.Category,
			&sr.Description, &sr.Vendor, &sr.Cost, &sr.Notes, &sr.VehicleName); err != nil {
			return nil, err
		}
		sr.Odometer = toI64(odo)
		out = append(out, sr)
	}
	return out, rows.Err()
}

// ---- reminders ----

func (s *Server) listRemindersByVehicle(vehicleID int64) ([]models.Reminder, error) {
	rows, err := s.db.Query(`
		SELECT id, vehicle_id, title, due_date, due_odometer, interval_months, interval_miles, notes, completed, notify, last_notified
		FROM reminders WHERE vehicle_id=? AND completed=0 ORDER BY COALESCE(due_date,'9999') ASC, id`, vehicleID)
	if err != nil {
		return nil, err
	}
	return scanReminders(rows)
}

// listAllReminders returns active reminders across all vehicles, annotated with
// their due status (using each vehicle's current odometer).
func (s *Server) listAllReminders() ([]models.Reminder, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.vehicle_id, r.title, r.due_date, r.due_odometer, r.interval_months,
		       r.interval_miles, r.notes, r.completed, r.notify, r.last_notified, v.name, v.odometer
		FROM reminders r JOIN vehicles v ON v.id = r.vehicle_id
		WHERE r.completed=0 AND v.archived=0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Reminder
	for rows.Next() {
		var r models.Reminder
		var dueDate, lastNotified sql.NullString
		var dueOdo, im, imi sql.NullInt64
		var completed, notify int
		var odo int64
		if err := rows.Scan(&r.ID, &r.VehicleID, &r.Title, &dueDate, &dueOdo, &im, &imi,
			&r.Notes, &completed, &notify, &lastNotified, &r.VehicleName, &odo); err != nil {
			return nil, err
		}
		r.DueDate = toStr(dueDate)
		r.DueOdometer = toI64(dueOdo)
		r.IntervalMonths = toInt(im)
		r.IntervalMiles = toInt(imi)
		r.Completed = completed != 0
		r.Notify = notify != 0
		r.LastNotified = toStr(lastNotified)
		annotateReminder(&r, odo)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Server) getReminder(id int64) (models.Reminder, error) {
	row := s.db.QueryRow(`
		SELECT id, vehicle_id, title, due_date, due_odometer, interval_months, interval_miles, notes, completed, notify, last_notified
		FROM reminders WHERE id=?`, id)
	var r models.Reminder
	var dueDate, lastNotified sql.NullString
	var dueOdo, im, imi sql.NullInt64
	var completed, notify int
	err := row.Scan(&r.ID, &r.VehicleID, &r.Title, &dueDate, &dueOdo, &im, &imi, &r.Notes, &completed, &notify, &lastNotified)
	if err != nil {
		return r, err
	}
	r.DueDate = toStr(dueDate)
	r.DueOdometer = toI64(dueOdo)
	r.IntervalMonths = toInt(im)
	r.IntervalMiles = toInt(imi)
	r.Completed = completed != 0
	r.Notify = notify != 0
	r.LastNotified = toStr(lastNotified)
	return r, nil
}

func scanReminders(rows *sql.Rows) ([]models.Reminder, error) {
	defer rows.Close()
	var out []models.Reminder
	for rows.Next() {
		var r models.Reminder
		var dueDate, lastNotified sql.NullString
		var dueOdo, im, imi sql.NullInt64
		var completed, notify int
		if err := rows.Scan(&r.ID, &r.VehicleID, &r.Title, &dueDate, &dueOdo, &im, &imi,
			&r.Notes, &completed, &notify, &lastNotified); err != nil {
			return nil, err
		}
		r.DueDate = toStr(dueDate)
		r.DueOdometer = toI64(dueOdo)
		r.IntervalMonths = toInt(im)
		r.IntervalMiles = toInt(imi)
		r.Completed = completed != 0
		r.Notify = notify != 0
		r.LastNotified = toStr(lastNotified)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Server) insertReminder(r models.Reminder) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO reminders(vehicle_id, title, due_date, due_odometer, interval_months, interval_miles, notes, notify)
		VALUES(?,?,?,?,?,?,?,?)`,
		r.VehicleID, r.Title, nullStr(r.DueDate), nullInt(r.DueOdometer),
		nullIntp(r.IntervalMonths), nullIntp(r.IntervalMiles), r.Notes, boolInt(r.Notify))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Server) updateReminderRow(r models.Reminder) error {
	// Editing resets the notification clock so a re-due reminder can fire again.
	_, err := s.db.Exec(`
		UPDATE reminders SET title=?, due_date=?, due_odometer=?, interval_months=?, interval_miles=?, notes=?, notify=?, last_notified=NULL
		WHERE id=?`,
		r.Title, nullStr(r.DueDate), nullInt(r.DueOdometer),
		nullIntp(r.IntervalMonths), nullIntp(r.IntervalMiles), r.Notes, boolInt(r.Notify), r.ID)
	return err
}

// markNotified records that a reminder's notification was sent on the given date.
func (s *Server) markNotified(id int64, date string) error {
	_, err := s.db.Exec(`UPDATE reminders SET last_notified=? WHERE id=?`, date, id)
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Server) completeReminderRow(id int64) error {
	_, err := s.db.Exec(`UPDATE reminders SET completed=1 WHERE id=?`, id)
	return err
}

func (s *Server) deleteReminderRow(id int64) error {
	_, err := s.db.Exec(`DELETE FROM reminders WHERE id=?`, id)
	return err
}

// ---- reference items ----

func scanReference(r scanner) (models.ReferenceItem, error) {
	var ri models.ReferenceItem
	err := r.Scan(&ri.ID, &ri.VehicleID, &ri.Kind, &ri.Name, &ri.PartNumber,
		&ri.Manufacturer, &ri.Capacity, &ri.Spec, &ri.Notes, &ri.Position)
	return ri, err
}

const refCols = `id, vehicle_id, kind, name, part_number, manufacturer, capacity, spec, notes, position`

func (s *Server) listReference(vehicleID int64) ([]models.ReferenceItem, error) {
	rows, err := s.db.Query(`SELECT `+refCols+`
		FROM reference_items WHERE vehicle_id=?
		ORDER BY kind, position, id`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ReferenceItem
	for rows.Next() {
		ri, err := scanReference(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ri)
	}
	return out, rows.Err()
}

func (s *Server) getReference(id int64) (models.ReferenceItem, error) {
	return scanReference(s.db.QueryRow(`SELECT `+refCols+` FROM reference_items WHERE id=?`, id))
}

func (s *Server) insertReference(ri models.ReferenceItem) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO reference_items(vehicle_id, kind, name, part_number, manufacturer, capacity, spec, notes)
		VALUES(?,?,?,?,?,?,?,?)`,
		ri.VehicleID, ri.Kind, ri.Name, ri.PartNumber, ri.Manufacturer, ri.Capacity, ri.Spec, ri.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Server) updateReferenceRow(ri models.ReferenceItem) error {
	_, err := s.db.Exec(`
		UPDATE reference_items SET kind=?, name=?, part_number=?, manufacturer=?, capacity=?, spec=?, notes=?
		WHERE id=?`,
		ri.Kind, ri.Name, ri.PartNumber, ri.Manufacturer, ri.Capacity, ri.Spec, ri.Notes, ri.ID)
	return err
}

func (s *Server) deleteReferenceRow(id int64) error {
	_, err := s.db.Exec(`DELETE FROM reference_items WHERE id=?`, id)
	return err
}

// ---- attachments ----

func (s *Server) insertAttachment(a models.Attachment) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO attachments(vehicle_id, service_id, kind, stored_name, original_name, content_type, size)
		VALUES(?,?,?,?,?,?,?)`,
		nullInt(a.VehicleID), nullInt(a.ServiceID), a.Kind, a.StoredName,
		a.OriginalName, a.ContentType, a.Size)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Server) getAttachment(id int64) (models.Attachment, error) {
	row := s.db.QueryRow(`
		SELECT id, vehicle_id, service_id, kind, stored_name, original_name, content_type, size
		FROM attachments WHERE id=?`, id)
	return scanAttachment(row)
}

func scanAttachment(r scanner) (models.Attachment, error) {
	var a models.Attachment
	var vid, sid sql.NullInt64
	err := r.Scan(&a.ID, &vid, &sid, &a.Kind, &a.StoredName, &a.OriginalName, &a.ContentType, &a.Size)
	if err != nil {
		return a, err
	}
	a.VehicleID = toI64(vid)
	a.ServiceID = toI64(sid)
	return a, nil
}

func (s *Server) listAttachmentsByVehicle(vehicleID int64, kind string) ([]models.Attachment, error) {
	rows, err := s.db.Query(`
		SELECT id, vehicle_id, service_id, kind, stored_name, original_name, content_type, size
		FROM attachments WHERE vehicle_id=? AND kind=? AND service_id IS NULL ORDER BY id DESC`, vehicleID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Attachment
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Server) listAttachmentsByService(serviceID int64) ([]models.Attachment, error) {
	rows, err := s.db.Query(`
		SELECT id, vehicle_id, service_id, kind, stored_name, original_name, content_type, size
		FROM attachments WHERE service_id=? ORDER BY id DESC`, serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Attachment
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Server) deleteAttachmentRow(id int64) error {
	_, err := s.db.Exec(`DELETE FROM attachments WHERE id=?`, id)
	return err
}

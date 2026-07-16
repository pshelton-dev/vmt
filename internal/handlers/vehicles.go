package handlers

// storedNamesForVehicle lists the stored filenames of every attachment tied to
// a vehicle (directly or via its service records), for cleanup on delete.
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

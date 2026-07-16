package handlers

import (
	"net/http"
	"time"
)

// apiReports returns the aggregate cost data the reports page charts from:
// grand totals, per-category and per-vehicle breakdowns, and a 12-month series.
func (s *Server) apiReports(w http.ResponseWriter, r *http.Request) {
	var total float64
	var count int
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(cost),0), COUNT(*) FROM service_records`).Scan(&total, &count)

	yearStart := time.Now().Format("2006") + "-01-01"
	var yearCost float64
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(cost),0) FROM service_records WHERE date >= ?`, yearStart).Scan(&yearCost)

	avg := 0.0
	if count > 0 {
		avg = total / float64(count)
	}

	byCategory := s.reportGroup(`
		SELECT 0, category, COALESCE(SUM(cost),0) FROM service_records
		WHERE cost > 0 GROUP BY category ORDER BY 3 DESC`, total)
	byVehicle := s.reportGroup(`
		SELECT v.id, v.name, COALESCE(SUM(sr.cost),0)
		FROM service_records sr JOIN vehicles v ON v.id = sr.vehicle_id
		WHERE sr.cost > 0 GROUP BY v.id ORDER BY 3 DESC`, total)

	keys, labels, values := s.monthlyTotals()
	monthly := make([]map[string]any, len(keys))
	for i := range keys {
		monthly[i] = map[string]any{"month": keys[i], "label": labels[i], "total": values[i]}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_cost":      total,
		"year_cost":       yearCost,
		"avg_per_service": avg,
		"service_count":   count,
		"by_category":     reportRowsJSON(byCategory),
		"by_vehicle":      reportRowsJSON(byVehicle),
		"monthly":         monthly,
	})
}

func reportRowsJSON(rows []reportRow) []map[string]any {
	out := make([]map[string]any, len(rows))
	for i, rr := range rows {
		out[i] = map[string]any{"id": rr.ID, "label": rr.Label, "total": rr.Total, "pct": rr.Pct}
	}
	return out
}

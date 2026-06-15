package handlers

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type reportRow struct {
	ID    int64
	Label string
	Total float64
	Pct   int
}

func (s *Server) removeStored(name string) {
	if name == "" {
		return
	}
	_ = os.Remove(filepath.Join(s.cfg.UploadDir, name))
}

func (s *Server) reports(w http.ResponseWriter, r *http.Request) {
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

	s.render(w, r, "reports", View{
		Title:  "Cost reports",
		Active: "reports",
		Data: map[string]any{
			"TotalCost":     total,
			"YearCost":      yearCost,
			"AvgPerService": avg,
			"ServiceCount":  count,
			"ByCategory":    byCategory,
			"ByVehicle":     byVehicle,
			"MonthlyChart":  s.monthlyChart(),
		},
	})
}

func (s *Server) reportGroup(query string, grand float64) []reportRow {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []reportRow
	for rows.Next() {
		var rr reportRow
		if err := rows.Scan(&rr.ID, &rr.Label, &rr.Total); err != nil {
			return out
		}
		if grand > 0 {
			rr.Pct = int(rr.Total/grand*100 + 0.5)
		}
		out = append(out, rr)
	}
	return out
}

// monthlyChart sums spending into the last 12 calendar-month buckets.
func (s *Server) monthlyChart() template.HTML {
	now := time.Now()
	type bucket struct {
		key   string
		label string
		total float64
	}
	buckets := make([]bucket, 12)
	index := map[string]int{}
	for i := 0; i < 12; i++ {
		m := now.AddDate(0, -(11 - i), 0)
		key := m.Format("2006-01")
		buckets[i] = bucket{key: key, label: m.Format("Jan")}
		index[key] = i
	}
	start := now.AddDate(0, -11, 0).Format("2006-01") + "-01"
	rows, err := s.db.Query(`
		SELECT substr(date,1,7) ym, COALESCE(SUM(cost),0)
		FROM service_records WHERE date >= ? GROUP BY ym`, start)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ym string
			var sum float64
			if rows.Scan(&ym, &sum) == nil {
				if i, ok := index[ym]; ok {
					buckets[i].total = sum
				}
			}
		}
	}
	labels := make([]string, 12)
	values := make([]float64, 12)
	any := false
	for i, b := range buckets {
		labels[i] = b.label
		values[i] = b.total
		if b.total > 0 {
			any = true
		}
	}
	if !any {
		return ""
	}
	cfg := s.cfg
	return barChart(labels, values, func(v float64) string { return money(cfg, v) })
}

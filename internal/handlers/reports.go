package handlers

import (
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

// reportGroup runs an aggregate query (id, label, total) and computes each
// row's share of the grand total.
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

// monthlyTotals sums spending into the last 12 calendar-month buckets,
// returning month keys (yyyy-mm), short labels and totals.
func (s *Server) monthlyTotals() (keys, labels []string, values []float64) {
	now := time.Now()
	keys = make([]string, 12)
	labels = make([]string, 12)
	values = make([]float64, 12)
	index := map[string]int{}
	for i := 0; i < 12; i++ {
		m := now.AddDate(0, -(11 - i), 0)
		keys[i] = m.Format("2006-01")
		labels[i] = m.Format("Jan")
		index[keys[i]] = i
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
					values[i] = sum
				}
			}
		}
	}
	return keys, labels, values
}

// removeStored deletes an uploaded file from disk (best-effort).
func (s *Server) removeStored(name string) {
	if name == "" {
		return
	}
	_ = os.Remove(filepath.Join(s.cfg.UploadDir, name))
}

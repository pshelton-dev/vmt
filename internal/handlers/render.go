package handlers

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"vmt/internal/config"
)

func (s *Server) funcMap() template.FuncMap {
	return template.FuncMap{
		"money":    money,
		"dist":     dist,
		"fmtdate":  fmtdate,
		"pint":     pint,
		"pstr":     pstr,
		"deref64":  deref64,
		"derefInt": derefInt,
		"years":    years,
	}
}

// years lists selectable model years, newest first, down to an early bound that
// still covers vintage vehicles.
func years() []int {
	const earliest = 1950
	top := time.Now().Year() + 1 // allow next model year
	out := make([]int, 0, top-earliest+1)
	for y := top; y >= earliest; y-- {
		out = append(out, y)
	}
	return out
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// money formats an amount with the configured currency symbol.
func money(cfg config.Config, v float64) string {
	return cfg.Currency + commaFloat(v)
}

// dist formats a distance with thousands separators and the configured unit.
func dist(cfg config.Config, n int64) string {
	return commaInt(n) + " " + cfg.DistanceUnit
}

// fmtdate parses an ISO date (yyyy-mm-dd, optionally with time) and formats it
// using the configured display layout.
func fmtdate(cfg config.Config, iso string) string {
	if iso == "" {
		return ""
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, iso); err == nil {
			return t.Format(cfg.DateFormat)
		}
	}
	return iso
}

func pint(p *int) string {
	if p == nil {
		return ""
	}
	return strconv.Itoa(*p)
}

func pstr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func deref64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func commaInt(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func commaFloat(v float64) string {
	whole := int64(v)
	frac := v - float64(whole)
	if v < 0 {
		frac = -frac
	}
	cents := int64(frac*100 + 0.5)
	if cents == 100 {
		cents = 0
		if whole >= 0 {
			whole++
		} else {
			whole--
		}
	}
	return commaInt(whole) + "." + fmt.Sprintf("%02d", cents)
}

func strip(s string) string { return strings.TrimSpace(s) }

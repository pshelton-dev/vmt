package handlers

import (
	"strconv"
	"strings"
	"time"
)

// optInt64 parses an optional integer field, returning nil when blank/invalid.
func optInt64(s string) *int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

// optDate validates an ISO date (yyyy-mm-dd), returning nil when blank/invalid.
func optDate(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return nil
	}
	return &s
}

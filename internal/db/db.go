// Package db opens the SQLite database and applies the schema.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Open opens (creating if needed) the SQLite database at path and applies the
// schema. The returned *sql.DB is ready for use.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// modernc/sqlite is fine with a single writer; cap connections to avoid
	// "database is locked" under concurrent writes.
	d.SetMaxOpenConns(1)
	d.SetConnMaxLifetime(time.Hour)

	if _, err := d.Exec(schema); err != nil {
		d.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if err := migrate(d); err != nil {
		d.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

// migrate applies idempotent column additions to databases created before a
// column existed. CREATE TABLE IF NOT EXISTS won't alter an existing table, so
// new columns are added here.
func migrate(d *sql.DB) error {
	type col struct{ table, name, def string }
	for _, c := range []col{
		{"reminders", "notify", "INTEGER NOT NULL DEFAULT 0"},
		{"reminders", "last_notified", "TEXT"},
	} {
		if err := addColumnIfMissing(d, c.table, c.name, c.def); err != nil {
			return err
		}
	}
	return nil
}

func addColumnIfMissing(d *sql.DB, table, name, def string) error {
	rows, err := d.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var colName, colType string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &colName, &colType, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if colName == name {
			return nil // already present
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = d.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, name, def))
	return err
}

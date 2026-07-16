// Package db opens the SQLite database and applies the schema.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// schemaVersion is stored in PRAGMA user_version. Bump it whenever schema.sql
// or migrate() changes; an existing database with an older version gets a
// pre-upgrade snapshot (VACUUM INTO <datadir>/backups/) before any schema
// change is applied, so a bad upgrade is always recoverable in place.
const schemaVersion = 1

// keepSnapshots is how many pre-upgrade snapshots to retain.
const keepSnapshots = 5

// Open opens (creating if needed) the SQLite database at path and applies the
// schema. The returned *sql.DB is ready for use.
func Open(path string) (*sql.DB, error) {
	_, statErr := os.Stat(path)
	preexisting := statErr == nil

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// modernc/sqlite is fine with a single writer; cap connections to avoid
	// "database is locked" under concurrent writes.
	d.SetMaxOpenConns(1)
	d.SetConnMaxLifetime(time.Hour)

	// Snapshot an existing database before touching its schema.
	if preexisting {
		if err := snapshotIfUpgrading(d, path); err != nil {
			d.Close()
			return nil, fmt.Errorf("pre-upgrade snapshot: %w", err)
		}
	}

	if _, err := d.Exec(schema); err != nil {
		d.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if err := migrate(d); err != nil {
		d.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	// PRAGMA doesn't take placeholders; schemaVersion is a trusted constant.
	if _, err := d.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion)); err != nil {
		d.Close()
		return nil, fmt.Errorf("set user_version: %w", err)
	}
	return d, nil
}

// snapshotIfUpgrading writes a consistent copy of the database to
// <datadir>/backups/pre-upgrade-<timestamp>.db when the stored schema version
// is older than schemaVersion, then prunes old snapshots. Failing to snapshot
// aborts startup: better to refuse an upgrade than to run one uninsured.
func snapshotIfUpgrading(d *sql.DB, path string) error {
	var v int
	if err := d.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= schemaVersion {
		return nil
	}
	dir := filepath.Join(filepath.Dir(path), "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	snap := filepath.Join(dir, "pre-upgrade-"+time.Now().Format("20060102-150405")+".db")
	if _, err := d.Exec(`VACUUM INTO ?`, snap); err != nil {
		return err
	}
	log.Printf("db: schema upgrade %d -> %d, snapshot saved to %s", v, schemaVersion, snap)
	pruneSnapshots(dir)
	return nil
}

// pruneSnapshots keeps the newest keepSnapshots pre-upgrade files. Best-effort:
// a failed prune only means extra snapshots lying around.
func pruneSnapshots(dir string) {
	matches, err := filepath.Glob(filepath.Join(dir, "pre-upgrade-*.db"))
	if err != nil {
		return
	}
	sort.Strings(matches) // timestamped names sort chronologically
	for _, old := range matches[:max(0, len(matches)-keepSnapshots)] {
		if err := os.Remove(old); err != nil {
			log.Printf("db: prune snapshot %s: %v", old, err)
		}
	}
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

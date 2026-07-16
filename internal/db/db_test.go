package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func snapshots(t *testing.T, dir string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, "backups", "pre-upgrade-*.db"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPreUpgradeSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vmt.db")

	// 1) A brand-new database is not an upgrade: no snapshot.
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO vehicles (name) VALUES ('truck')`); err != nil {
		t.Fatal(err)
	}
	d.Close()
	if got := snapshots(t, dir); len(got) != 0 {
		t.Fatalf("fresh db: expected no snapshots, got %v", got)
	}

	// 2) Reopening at the same schema version: still no snapshot.
	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	if got := snapshots(t, dir); len(got) != 0 {
		t.Fatalf("same version: expected no snapshots, got %v", got)
	}

	// 3) A database from an older schema version (like every pre-existing prod
	// db, which has user_version 0) gets snapshotted before schema is applied.
	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec("PRAGMA user_version = 0"); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	got := snapshots(t, dir)
	if len(got) != 1 {
		t.Fatalf("upgrade: expected 1 snapshot, got %v", got)
	}

	// The snapshot is a valid SQLite db containing the pre-upgrade data.
	sd, err := Open(got[0])
	if err != nil {
		t.Fatalf("snapshot not openable: %v", err)
	}
	defer sd.Close()
	var n int
	if err := sd.QueryRow(`SELECT COUNT(*) FROM vehicles`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("snapshot data: want 1 vehicle, got %d (err=%v)", n, err)
	}
}

func TestPruneSnapshots(t *testing.T) {
	dir := t.TempDir()
	bdir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(bdir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < keepSnapshots+3; i++ {
		name := filepath.Join(bdir, fmt.Sprintf("pre-upgrade-2026010%d-000000.db", i))
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pruneSnapshots(bdir)
	left, _ := filepath.Glob(filepath.Join(bdir, "pre-upgrade-*.db"))
	if len(left) != keepSnapshots {
		t.Fatalf("want %d snapshots after prune, got %d", keepSnapshots, len(left))
	}
	// The newest ones survive.
	for _, f := range left {
		if filepath.Base(f) < fmt.Sprintf("pre-upgrade-2026010%d-000000.db", 3) {
			t.Fatalf("old snapshot survived prune: %s", f)
		}
	}
}

package handlers

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxRestore = 2 << 30 // 2 GiB

// backup streams a gzipped tar containing a consistent snapshot of the database
// plus all uploaded files.
func (s *Server) backup(w http.ResponseWriter, r *http.Request) {
	// VACUUM INTO produces a single clean DB file with the WAL already merged,
	// giving a transactionally-consistent snapshot without locking out writes.
	snap := filepath.Join(s.cfg.DataDir, fmt.Sprintf(".snapshot-%s.db", randName()))
	if _, err := s.db.Exec(`VACUUM INTO ?`, snap); err != nil {
		apiError(w, http.StatusInternalServerError, "could not snapshot database")
		return
	}
	defer os.Remove(snap)

	name := "vmt-backup-" + time.Now().Format("20060102-150405") + ".tar.gz"
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))

	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	if err := addFileToTar(tw, snap, "vmt.db"); err != nil {
		log.Printf("backup: add db: %v", err)
		return
	}
	// Add uploads under uploads/.
	uploads := s.cfg.UploadDir
	_ = filepath.Walk(uploads, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(uploads, path)
		if rerr != nil {
			return nil
		}
		if e := addFileToTar(tw, path, "uploads/"+filepath.ToSlash(rel)); e != nil {
			log.Printf("backup: add %s: %v", rel, e)
		}
		return nil
	})
}

func addFileToTar(tw *tar.Writer, path, name string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name:    name,
		Mode:    0o644,
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

// restore replaces the database and uploads from an uploaded backup archive,
// then restarts the process so everything re-opens against the restored data.
func (s *Server) restore(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		apiError(w, http.StatusBadRequest, "upload failed")
		return
	}
	file, _, err := r.FormFile("backup")
	if err != nil {
		apiError(w, http.StatusBadRequest, "choose a backup file to restore (multipart \"backup\" field)")
		return
	}
	defer file.Close()

	// Stage inside the data dir so the final move stays on one filesystem.
	stage := filepath.Join(s.cfg.DataDir, ".restore-tmp")
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		apiError(w, http.StatusInternalServerError, "could not stage restore")
		return
	}
	defer os.RemoveAll(stage)

	if err := extractArchive(io.LimitReader(file, maxRestore), stage); err != nil {
		apiError(w, http.StatusBadRequest, "invalid backup archive: "+err.Error())
		return
	}

	stagedDB := filepath.Join(stage, "vmt.db")
	if err := validateDB(stagedDB); err != nil {
		apiError(w, http.StatusBadRequest, "backup does not contain a valid database")
		return
	}

	if err := s.swapInRestore(stage, stagedDB); err != nil {
		// The pre-restore copies (.prerestore) are left in place for recovery.
		apiError(w, http.StatusInternalServerError, "restore failed: "+err.Error())
		return
	}

	// Respond before exiting; the container's restart policy brings us back up
	// against the restored data. The SPA shows a "restarting" note and reloads.
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarting"})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	log.Print("restore complete — restarting to load restored data")
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
}

// swapInRestore moves the staged DB and uploads into place, keeping the
// previous data as *.prerestore for recovery.
func (s *Server) swapInRestore(stage, stagedDB string) error {
	dbPath := s.cfg.DBPath

	// Release our handle so files can be replaced cleanly.
	_ = s.db.Close()

	// Preserve the current DB, then drop stale WAL/SHM siblings.
	_ = os.Remove(dbPath + ".prerestore")
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Rename(dbPath, dbPath+".prerestore"); err != nil {
			return err
		}
	}
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	if err := os.Rename(stagedDB, dbPath); err != nil {
		return err
	}

	// Replace uploads if the archive carried them.
	stagedUploads := filepath.Join(stage, "uploads")
	if fi, err := os.Stat(stagedUploads); err == nil && fi.IsDir() {
		_ = os.RemoveAll(s.cfg.UploadDir + ".prerestore")
		if _, err := os.Stat(s.cfg.UploadDir); err == nil {
			_ = os.Rename(s.cfg.UploadDir, s.cfg.UploadDir+".prerestore")
		}
		if err := os.Rename(stagedUploads, s.cfg.UploadDir); err != nil {
			return err
		}
	}
	return nil
}

// extractArchive unpacks a gzipped tar into dst, accepting only the expected
// entries (vmt.db and files under uploads/) and rejecting path traversal.
func extractArchive(r io.Reader, dst string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("not a gzip file")
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	sawDB := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("corrupt archive")
		}
		clean := filepath.Clean(filepath.FromSlash(hdr.Name))
		if clean == "vmt.db" {
			// ok
		} else if !strings.HasPrefix(clean, "uploads"+string(os.PathSeparator)) {
			continue // ignore anything unexpected
		}
		if strings.Contains(clean, "..") {
			return fmt.Errorf("unsafe path in archive")
		}
		target := filepath.Join(dst, clean)
		if hdr.FileInfo().IsDir() {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, io.LimitReader(tr, maxRestore)); err != nil {
			out.Close()
			return err
		}
		out.Close()
		if clean == "vmt.db" {
			sawDB = true
		}
	}
	if !sawDB {
		return fmt.Errorf("missing vmt.db")
	}
	return nil
}

// validateDB opens the file read-only and confirms it is a usable SQLite DB.
func validateDB(path string) error {
	d, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return err
	}
	defer d.Close()
	var n int
	return d.QueryRow(`SELECT count(*) FROM sqlite_master`).Scan(&n)
}

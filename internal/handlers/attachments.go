package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"vmt/internal/models"
)

const maxUpload = 32 << 20 // 32 MiB

func randName() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// saveUpload stores an uploaded file (if present) and inserts an attachment row.
// It returns the new attachment ID, or 0 if no file was submitted.
func (s *Server) saveUpload(r *http.Request, field, kind string, vehicleID, serviceID *int64) (int64, error) {
	file, hdr, err := r.FormFile(field)
	if err != nil {
		return 0, nil // no file submitted
	}
	defer file.Close()

	if err := os.MkdirAll(s.cfg.UploadDir, 0o755); err != nil {
		return 0, err
	}
	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	stored := randName() + ext
	dst, err := os.Create(filepath.Join(s.cfg.UploadDir, stored))
	if err != nil {
		return 0, err
	}
	size, err := io.Copy(dst, io.LimitReader(file, maxUpload))
	closeErr := dst.Close()
	if err != nil {
		os.Remove(filepath.Join(s.cfg.UploadDir, stored))
		return 0, err
	}
	if closeErr != nil {
		return 0, closeErr
	}

	ct := hdr.Header.Get("Content-Type")
	if ct == "" {
		ct = mime.TypeByExtension(ext)
	}
	a := models.Attachment{
		VehicleID:    vehicleID,
		ServiceID:    serviceID,
		Kind:         kind,
		StoredName:   stored,
		OriginalName: filepath.Base(hdr.Filename),
		ContentType:  ct,
		Size:         size,
	}
	return s.insertAttachment(a)
}

// serveFile streams a stored attachment to the client.
func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a, err := s.getAttachment(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.cfg.UploadDir, a.StoredName)
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if a.ContentType != "" {
		w.Header().Set("Content-Type", a.ContentType)
	}
	if a.Kind != "photo" {
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", a.OriginalName))
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, a.OriginalName, fi.ModTime(), f)
}

func (s *Server) uploadPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad vehicle id")
		return
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		s.setFlash(w, "Upload too large.")
		redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
		return
	}
	aid, err := s.saveUpload(r, "photo", "photo", &id, nil)
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save photo")
		return
	}
	if aid > 0 {
		// If the vehicle has no primary photo yet, make this one primary.
		if v, err := s.getVehicle(id); err == nil && v.PhotoID == nil {
			_ = s.setVehiclePhoto(id, aid)
		}
		s.setFlash(w, "Photo uploaded.")
	}
	redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
}

func (s *Server) uploadDocument(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad vehicle id")
		return
	}
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		s.setFlash(w, "Upload too large.")
		redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
		return
	}
	if _, err := s.saveUpload(r, "document", "document", &id, nil); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save document")
		return
	}
	s.setFlash(w, "Document uploaded.")
	redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
}

func (s *Server) setPrimaryPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad vehicle id")
		return
	}
	aid, err := pathID(r, "aid")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad photo id")
		return
	}
	_ = s.setVehiclePhoto(id, aid)
	redirect(w, r, fmt.Sprintf("/vehicles/%d", id))
}

func (s *Server) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		s.renderError(w, r, http.StatusBadRequest, "bad id")
		return
	}
	a, err := s.getAttachment(id)
	if err != nil {
		s.renderError(w, r, http.StatusNotFound, "not found")
		return
	}
	if err := s.deleteAttachmentRow(id); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not delete")
		return
	}
	_ = os.Remove(filepath.Join(s.cfg.UploadDir, a.StoredName))
	s.setFlash(w, "Deleted.")
	dest := "/"
	if a.VehicleID != nil {
		dest = fmt.Sprintf("/vehicles/%d", *a.VehicleID)
	} else if a.ServiceID != nil {
		if sr, err := s.getService(*a.ServiceID); err == nil {
			dest = fmt.Sprintf("/vehicles/%d", sr.VehicleID)
		}
	}
	redirect(w, r, refererOr(r, dest))
}

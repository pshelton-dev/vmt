package handlers

import (
	"net/http"

	"vmt/internal/auth"
)

func (s *Server) isAuthed(r *http.Request) bool {
	c, err := r.Cookie(auth.CookieName())
	if err != nil {
		return false
	}
	return s.auth.Valid(c.Value)
}

func secureCookie(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) error {
	token, expires, err := s.auth.CreateSession()
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName(),
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   secureCookie(r),
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

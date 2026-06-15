package handlers

import (
	"net/http"
	"time"

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

// requireAuth wraps a handler, redirecting unauthenticated requests to the
// login (or first-run setup) page.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		configured, err := s.auth.IsConfigured()
		if err != nil {
			s.renderError(w, r, http.StatusInternalServerError, "database error")
			return
		}
		if !configured {
			redirect(w, r, "/setup")
			return
		}
		if !s.isAuthed(r) {
			redirect(w, r, "/login")
			return
		}
		next(w, r)
	}
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

func (s *Server) getLogin(w http.ResponseWriter, r *http.Request) {
	configured, _ := s.auth.IsConfigured()
	if !configured {
		redirect(w, r, "/setup")
		return
	}
	if s.isAuthed(r) {
		redirect(w, r, "/")
		return
	}
	s.render(w, r, "login", View{Title: "Sign in"})
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	ok, err := s.auth.Check(r.FormValue("password"))
	if err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "login failed")
		return
	}
	if !ok {
		s.setFlash(w, "Incorrect password.")
		redirect(w, r, "/login")
		return
	}
	if err := s.startSession(w, r); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not start session")
		return
	}
	redirect(w, r, "/")
}

func (s *Server) getSetup(w http.ResponseWriter, r *http.Request) {
	if configured, _ := s.auth.IsConfigured(); configured {
		redirect(w, r, "/login")
		return
	}
	s.render(w, r, "setup", View{Title: "Welcome"})
}

func (s *Server) postSetup(w http.ResponseWriter, r *http.Request) {
	if configured, _ := s.auth.IsConfigured(); configured {
		redirect(w, r, "/login")
		return
	}
	pw := r.FormValue("password")
	if len(pw) < 6 || pw != r.FormValue("confirm") {
		s.setFlash(w, "Passwords must match and be at least 6 characters.")
		redirect(w, r, "/setup")
		return
	}
	if err := s.auth.SetPassword(pw); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not save password")
		return
	}
	if err := s.startSession(w, r); err != nil {
		s.renderError(w, r, http.StatusInternalServerError, "could not start session")
		return
	}
	redirect(w, r, "/")
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName()); err == nil {
		s.auth.Destroy(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.CookieName(), Value: "", Path: "/", Expires: time.Unix(0, 0), MaxAge: -1,
	})
	redirect(w, r, "/login")
}

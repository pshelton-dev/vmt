package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"html"
	"log"
	"net/http"

	"vmt/internal/mail"
)

// prefOAuthState holds the one-shot CSRF token for an in-flight Google consent
// flow. It is written when the flow starts and cleared as soon as the callback
// consumes it, so a stale or replayed callback cannot be accepted.
const prefOAuthState = "gmail_oauth_state"

// apiGoogleOAuthStart redirects the browser to Google's consent screen.
// Client ID/secret must already be saved in Settings.
func (s *Server) apiGoogleOAuthStart(w http.ResponseWriter, r *http.Request) {
	g := s.gmailSettings()
	if g.ClientID == "" || g.ClientSecret == "" {
		apiError(w, http.StatusBadRequest, "save the Google client ID and secret first")
		return
	}
	redirect, err := s.oauthRedirectURI()
	if err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := randomToken()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not start the flow")
		return
	}
	if err := s.setPref(prefOAuthState, state); err != nil {
		apiError(w, http.StatusInternalServerError, "could not start the flow")
		return
	}
	http.Redirect(w, r, mail.AuthCodeURL(g.ClientID, redirect, state), http.StatusFound)
}

// apiGoogleOAuthCallback receives Google's redirect, exchanges the code for a
// refresh token, and stores it. It answers with an HTML page rather than JSON
// because the browser lands here directly from Google.
func (s *Server) apiGoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		s.oauthResult(w, "Google declined the request: "+e)
		return
	}
	want := s.getPref(prefOAuthState, "")
	got := q.Get("state")
	// Single-use: clear the state before doing anything with the code.
	_ = s.setPref(prefOAuthState, "")
	if want == "" || got == "" || got != want {
		s.oauthResult(w, "This authorization link is stale or was already used. Start again from Settings.")
		return
	}
	code := q.Get("code")
	if code == "" {
		s.oauthResult(w, "Google did not return an authorization code.")
		return
	}
	redirect, err := s.oauthRedirectURI()
	if err != nil {
		s.oauthResult(w, err.Error())
		return
	}
	g := s.gmailSettings()
	refresh, email, err := mail.ExchangeCode(g.ClientID, g.ClientSecret, redirect, code)
	if err != nil {
		log.Printf("oauth: exchange failed: %v", err)
		s.oauthResult(w, "Could not complete the connection: "+err.Error())
		return
	}
	_ = s.setPref(prefGmailRefreshToken, refresh)
	_ = s.setPref(prefGmailEmail, email)
	// Connecting an account is an explicit choice to use it.
	_ = s.setPref(prefMailProvider, providerGmail)
	log.Printf("oauth: connected Google account %s", email)
	s.oauthResult(w, "")
}

// apiGoogleDisconnect forgets the stored Google tokens and falls back to SMTP.
func (s *Server) apiGoogleDisconnect(w http.ResponseWriter, r *http.Request) {
	_ = s.setPref(prefGmailRefreshToken, "")
	_ = s.setPref(prefGmailEmail, "")
	if s.mailProvider() == providerGmail {
		_ = s.setPref(prefMailProvider, providerSMTP)
	}
	w.WriteHeader(http.StatusNoContent)
}

// oauthResult renders a minimal end-of-flow page. An empty errMsg means success.
func (s *Server) oauthResult(w http.ResponseWriter, errMsg string) {
	title, detail := "Google account connected", "You can close this tab and return to VMT."
	if errMsg != "" {
		title, detail = "Connection failed", errMsg
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Self-contained page: the SPA isn't running in this tab.
	_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>` + html.EscapeString(title) + `</title>` +
		`<style>body{font-family:system-ui,sans-serif;margin:0;min-height:100vh;display:flex;
		align-items:center;justify-content:center;background:#0f1115;color:#e6e8ec}
		.card{max-width:32rem;padding:2rem;border:1px solid #262a33;border-radius:12px;background:#151922}
		h1{font-size:1.25rem;margin:0 0 .5rem}p{margin:0;color:#9aa3b2;line-height:1.5}
		a{color:#5b8cff}</style>` +
		`<div class="card"><h1>` + html.EscapeString(title) + `</h1><p>` + html.EscapeString(detail) +
		`</p><p style="margin-top:1rem"><a href="/settings">Back to VMT settings</a></p></div>`))
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

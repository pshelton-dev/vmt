package handlers

import (
	"fmt"
	"strings"

	"vmt/internal/mail"
)

// Mail settings live in the `settings` table alongside display preferences, so
// everything is configurable from the app itself — there are deliberately no
// VMT_SMTP_* environment variables any more (removed in v2.2.0).
const (
	prefMailProvider = "mail_provider" // "smtp" | "gmail"

	prefSMTPHost     = "smtp_host"
	prefSMTPPort     = "smtp_port"
	prefSMTPUser     = "smtp_user"
	prefSMTPPass     = "smtp_pass"
	prefSMTPFrom     = "smtp_from"
	prefSMTPTLS      = "smtp_tls"
	prefSMTPInsecure = "smtp_insecure"

	prefGmailClientID     = "gmail_client_id"
	prefGmailClientSecret = "gmail_client_secret"
	prefGmailRefreshToken = "gmail_refresh_token"
	prefGmailEmail        = "gmail_email" // connected account, for display + From
)

// providerGmail / providerSMTP are the valid mail_provider values.
const (
	providerSMTP  = "smtp"
	providerGmail = "gmail"
)

// mailProvider returns the configured provider, defaulting to SMTP.
func (s *Server) mailProvider() string {
	if s.getPref(prefMailProvider, providerSMTP) == providerGmail {
		return providerGmail
	}
	return providerSMTP
}

// smtpSettings builds the SMTP config from stored preferences.
func (s *Server) smtpSettings() mail.SMTP {
	return mail.SMTP{
		Host:     s.getPref(prefSMTPHost, ""),
		Port:     s.getPref(prefSMTPPort, "587"),
		User:     s.getPref(prefSMTPUser, ""),
		Pass:     s.getPref(prefSMTPPass, ""),
		From:     s.getPref(prefSMTPFrom, ""),
		TLS:      s.getPref(prefSMTPTLS, "starttls"),
		Insecure: s.getPref(prefSMTPInsecure, "0") == "1",
	}
}

// gmailSettings builds the Gmail OAuth config from stored preferences. The
// connected account address doubles as the From, since Gmail rejects sends
// claiming to be from anyone else.
func (s *Server) gmailSettings() mail.Gmail {
	return mail.Gmail{
		ClientID:     s.getPref(prefGmailClientID, ""),
		ClientSecret: s.getPref(prefGmailClientSecret, ""),
		RefreshToken: s.getPref(prefGmailRefreshToken, ""),
		From:         s.getPref(prefGmailEmail, ""),
	}
}

// mailConfigured reports whether the active provider has enough settings to
// attempt a send.
func (s *Server) mailConfigured() bool {
	if s.mailProvider() == providerGmail {
		return s.gmailSettings().Configured()
	}
	return s.smtpSettings().Configured()
}

// mailFrom is the address the active provider sends as (for display).
func (s *Server) mailFrom() string {
	if s.mailProvider() == providerGmail {
		return s.getPref(prefGmailEmail, "")
	}
	return s.getPref(prefSMTPFrom, "")
}

// sendMail delivers a message via whichever provider is currently selected.
// This is the single outbound-mail entry point for the whole app.
func (s *Server) sendMail(to, subject, body string) error {
	switch s.mailProvider() {
	case providerGmail:
		return mail.SendGmail(s.gmailSettings(), to, subject, body)
	default:
		return mail.Send(s.smtpSettings(), to, subject, body)
	}
}

// oauthRedirectURI is the callback Google sends the browser back to. It must
// match a URI registered on the OAuth client exactly, so it is derived from the
// configured public base URL.
func (s *Server) oauthRedirectURI() (string, error) {
	base := strings.TrimRight(s.currentCfg().BaseURL, "/")
	if base == "" {
		return "", fmt.Errorf("VMT_BASE_URL is not set, so the OAuth redirect URI cannot be built")
	}
	return base + "/api/v1/oauth/google/callback", nil
}

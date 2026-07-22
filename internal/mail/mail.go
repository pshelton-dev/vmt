// Package mail sends outbound notification email, either over SMTP or through
// the Gmail API. The SMTP path uses only the standard library and supports
// STARTTLS (default), implicit TLS, and plaintext, and can optionally skip TLS
// certificate verification (self-signed servers).
//
// Settings live in the database (see internal/handlers), not the environment,
// so these structs are populated per-send from stored preferences.
package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

const dialTimeout = 15 * time.Second

// SMTP holds outbound mail server settings.
type SMTP struct {
	Host string
	Port string
	User string
	Pass string
	From string
	TLS  string // "starttls" (default), "implicit", or "none"
	// Insecure skips TLS certificate verification (for mail servers with a
	// self-signed cert). Applies to STARTTLS and implicit TLS.
	Insecure bool
}

// Configured reports whether enough SMTP settings are present to send mail.
func (s SMTP) Configured() bool {
	return s.Host != "" && s.From != ""
}

// Send delivers a plaintext message to a single recipient over SMTP.
func Send(cfg SMTP, to, subject, body string) error {
	if !cfg.Configured() {
		return fmt.Errorf("SMTP is not configured")
	}
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	msg := buildMessage(cfg.From, to, subject, body)

	// TLS settings shared by the STARTTLS and implicit-TLS paths. InsecureSkipVerify
	// is opt-in (VMT_SMTP_INSECURE) for mail servers with self-signed certs.
	tlsConf := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.Insecure,
	}

	var auth smtp.Auth
	if cfg.User != "" {
		auth = smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	}

	switch strings.ToLower(cfg.TLS) {
	case "implicit":
		return sendImplicit(addr, cfg.Host, tlsConf, auth, cfg.From, to, msg)
	case "none":
		return sendPlain(addr, cfg.Host, nil, auth, cfg.From, to, msg)
	default: // "starttls"
		return sendPlain(addr, cfg.Host, tlsConf, auth, cfg.From, to, msg)
	}
}

// sendPlain dials plaintext and, when tlsConf is non-nil, upgrades via STARTTLS
// before authenticating and sending.
func sendPlain(addr, host string, tlsConf *tls.Config, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if tlsConf != nil {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return fmt.Errorf("server does not support STARTTLS")
		}
		if err := c.StartTLS(tlsConf); err != nil {
			return err
		}
	}
	return finishSend(c, auth, from, to, msg)
}

// sendImplicit dials with TLS from the first byte (port 465).
func sendImplicit(addr, host string, tlsConf *tls.Config, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", addr, tlsConf)
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	return finishSend(c, auth, from, to, msg)
}

func finishSend(c *smtp.Client, auth smtp.Auth, from, to string, msg []byte) error {
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	// Normalise line endings to CRLF for SMTP.
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return []byte(b.String())
}

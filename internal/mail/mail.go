// Package mail sends outbound notification email over SMTP using only the
// standard library. It supports STARTTLS (default), implicit TLS, and plaintext.
package mail

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"vmt/internal/config"
)

// Send delivers a plaintext message to a single recipient.
func Send(cfg config.SMTP, to, subject, body string) error {
	if !cfg.Configured() {
		return fmt.Errorf("SMTP is not configured")
	}
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	msg := buildMessage(cfg.From, to, subject, body)

	var auth smtp.Auth
	if cfg.User != "" {
		auth = smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	}

	switch strings.ToLower(cfg.TLS) {
	case "implicit":
		return sendImplicitTLS(addr, cfg.Host, auth, cfg.From, to, msg)
	case "none":
		return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg)
	default: // "starttls"
		return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg)
	}
}

// sendImplicitTLS handles servers that expect TLS from the first byte (port 465).
func sendImplicitTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
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
